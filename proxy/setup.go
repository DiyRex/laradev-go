package proxy

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// SetupProxy performs the one-time setup for a project's HTTPS proxy:
//  1. Generates the local CA (pure Go, no external tools)
//  2. Trusts the CA in the OS keychain
//  3. Generates a TLS certificate for the domain (pure Go)
//  4. Adds the domain to /etc/hosts
//  5. Saves proxy.conf to ~/.laradev/projects/{hash}/
func SetupProxy(cfg *ProxyConfig) error {
	fmt.Printf("  Setting up HTTPS proxy for %s…\n", cfg.Domain)
	fmt.Println()

	// 1. Ensure global dirs
	if err := EnsureGlobalDirs(); err != nil {
		return fmt.Errorf("cannot create ~/.laradev dirs: %w", err)
	}

	// 2 & 3. CA generation + trust (always re-trust so browsers accept the cert).
	if err := TrustCA(); err != nil {
		// Non-fatal — proxy still works, browser will just show a warning.
		fmt.Printf("  ⚠  CA trust failed: %v\n", err)
		fmt.Printf("  ⚠  To trust manually run:\n")
		printManualTrustHint()
	}

	// 4. Domain certificate (signed by local CA)
	if err := GenerateDomainCert(cfg.Domain); err != nil {
		return fmt.Errorf("certificate generation failed: %w", err)
	}

	// 5. /etc/hosts entry
	if err := AddHostsEntry(cfg.Domain); err != nil {
		return fmt.Errorf("failed to update /etc/hosts: %w", err)
	}

	// 6. Persistent port forwarding (443→8443) so no :8443 in the URL.
	//    Skip if port 443 is already occupied by another process (e.g. Docker Desktop).
	fmt.Println("  Setting up port 443 → 8443 forwarding (sudo required)…")
	if port443InUse() {
		fmt.Println("  ⚠  Port 443 is already in use by another process (Docker Desktop, nginx, etc.)")
		fmt.Printf("  ⚠  App will be at https://%s:%s\n", cfg.Domain, cfg.ProxyPort)
		fmt.Println("  ⚠  Stop the conflicting process and run: laradev proxy:ports")
	} else if err := SetupPersistentPortForwarding(cfg); err != nil {
		fmt.Printf("  ⚠  Port forwarding failed: %v\n", err)
		fmt.Printf("  ⚠  App will be at https://%s:%s (retry with: laradev proxy:ports)\n",
			cfg.Domain, cfg.ProxyPort)
	} else {
		cfg.PortForwarding = true
		fmt.Println("  ✓ Port forwarding configured (persists across reboots)")
	}

	// 7. Save config
	cfg.Enabled = true
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("cannot save proxy config: %w", err)
	}

	fmt.Println()
	fmt.Println("  ✓ Done! Proxy will start automatically next time you run laradev.")
	fmt.Printf("  ✓ Your app will be at: %s\n", cfg.AppURL())
	fmt.Println()

	return nil
}

// AddHostsEntry appends "127.0.0.1 <domain> # laradev" to /etc/hosts if not present.
func AddHostsEntry(domain string) error {
	hostsFile := "/etc/hosts"

	data, err := os.ReadFile(hostsFile)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", hostsFile, err)
	}
	if strings.Contains(string(data), domain) {
		fmt.Printf("  ✓ /etc/hosts already has entry for %s\n", domain)
		return nil
	}

	entry := fmt.Sprintf("\n127.0.0.1 %s # laradev\n", domain)

	// Try direct write first.
	f, err := os.OpenFile(hostsFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()
		if _, werr := f.WriteString(entry); werr != nil {
			return werr
		}
		fmt.Printf("  ✓ Added to /etc/hosts: 127.0.0.1 %s\n", domain)
		return nil
	}

	// Fall back to sudo.
	fmt.Printf("  Adding 127.0.0.1 %s to /etc/hosts (sudo required)…\n", domain)
	cmd := exec.Command("sudo", "sh", "-c",
		fmt.Sprintf(`printf "\n127.0.0.1 %s # laradev\n" >> /etc/hosts`, domain))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sudo /etc/hosts write failed: %w", err)
	}
	fmt.Printf("  ✓ Added to /etc/hosts: 127.0.0.1 %s\n", domain)
	return nil
}

// RemoveHostsEntry removes the laradev-managed line for domain from /etc/hosts.
func RemoveHostsEntry(domain string) error {
	hostsFile := "/etc/hosts"
	data, err := os.ReadFile(hostsFile)
	if err != nil {
		return err
	}

	var kept []string
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, domain) && strings.Contains(line, "laradev") {
			continue
		}
		kept = append(kept, line)
	}
	cleaned := strings.Join(kept, "\n")

	if err := os.WriteFile(hostsFile, []byte(cleaned), 0644); err != nil {
		cmd := exec.Command("sudo", "sh", "-c",
			fmt.Sprintf(`grep -v "%s.*laradev" /etc/hosts > /tmp/.laradev_hosts && sudo mv /tmp/.laradev_hosts /etc/hosts`, domain))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		return cmd.Run()
	}
	return nil
}

// SetupPersistentPortForwarding sets up port 443→ProxyPort and 80→HTTPPort in a way
// that survives reboots.
//   - macOS: writes a LaunchDaemon plist and loads it immediately
//   - Linux: writes a systemd service and enables + starts it
func SetupPersistentPortForwarding(cfg *ProxyConfig) error {
	switch runtime.GOOS {
	case "darwin":
		return setupLaunchDaemon(cfg)
	case "linux":
		return setupSystemdPortForward(cfg)
	}
	return nil
}

const launchDaemonLabel = "com.laradev.portforward"
const launchDaemonPath = "/Library/LaunchDaemons/com.laradev.portforward.plist"

func setupLaunchDaemon(cfg *ProxyConfig) error {
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>/bin/sh</string>
        <string>-c</string>
        <string>printf "rdr pass on lo0 proto tcp from any to any port 443 -> 127.0.0.1 port %s\nrdr pass on lo0 proto tcp from any to any port 80 -> 127.0.0.1 port %s\n" | pfctl -ef - 2&gt;/dev/null; sysctl -w net.inet.ip.forwarding=1 2&gt;/dev/null; true</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>StandardErrorPath</key>
    <string>/dev/null</string>
    <key>StandardOutPath</key>
    <string>/dev/null</string>
</dict>
</plist>`, launchDaemonLabel, cfg.ProxyPort, cfg.HTTPPort)

	// Write plist via sudo
	writeCmd := exec.Command("sudo", "sh", "-c",
		fmt.Sprintf("cat > %s", launchDaemonPath))
	writeCmd.Stdin = strings.NewReader(plist)
	writeCmd.Stdout = os.Stdout
	writeCmd.Stderr = os.Stderr
	if err := writeCmd.Run(); err != nil {
		return fmt.Errorf("write LaunchDaemon: %w", err)
	}

	// Unload if already loaded (ignore error — may not exist yet)
	_ = exec.Command("sudo", "launchctl", "unload", launchDaemonPath).Run()

	// Load and start the daemon
	loadCmd := exec.Command("sudo", "launchctl", "load", "-w", launchDaemonPath)
	loadCmd.Stdout = os.Stdout
	loadCmd.Stderr = os.Stderr
	if err := loadCmd.Run(); err != nil {
		return fmt.Errorf("load LaunchDaemon: %w", err)
	}
	return nil
}

const systemdServicePath = "/etc/systemd/system/laradev-portforward.service"

func setupSystemdPortForward(cfg *ProxyConfig) error {
	unit := fmt.Sprintf(`[Unit]
Description=LaraDev port forwarding (443->%s, 80->%s)
After=network.target

[Service]
Type=oneshot
ExecStart=/bin/sh -c "iptables -t nat -C OUTPUT -p tcp --dport 443 -j REDIRECT --to-port %s 2>/dev/null || iptables -t nat -A OUTPUT -p tcp --dport 443 -j REDIRECT --to-port %s; iptables -t nat -C OUTPUT -p tcp --dport 80 -j REDIRECT --to-port %s 2>/dev/null || iptables -t nat -A OUTPUT -p tcp --dport 80 -j REDIRECT --to-port %s"
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
`, cfg.ProxyPort, cfg.HTTPPort,
		cfg.ProxyPort, cfg.ProxyPort,
		cfg.HTTPPort, cfg.HTTPPort)

	tmp, err := os.CreateTemp("", "laradev-portforward-*.service")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(unit); err != nil {
		return err
	}
	tmp.Close()

	for _, c := range [][]string{
		{"sudo", "cp", tmp.Name(), systemdServicePath},
		{"sudo", "systemctl", "daemon-reload"},
		{"sudo", "systemctl", "enable", "--now", filepath.Base(systemdServicePath)},
	} {
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s: %w", c[1], err)
		}
	}
	return nil
}

// SetupPortForwarding applies OS-level port redirection 443→ProxyPort and 80→HTTPPort.
func SetupPortForwarding(cfg *ProxyConfig) error {
	switch runtime.GOOS {
	case "darwin":
		return applyPfctl(cfg)
	case "linux":
		return applyIptables(cfg)
	}
	return nil
}

func applyPfctl(cfg *ProxyConfig) error {
	rules := fmt.Sprintf(
		"rdr pass on lo0 proto tcp from any to any port 443 -> 127.0.0.1 port %s\n"+
			"rdr pass on lo0 proto tcp from any to any port 80 -> 127.0.0.1 port %s",
		cfg.ProxyPort, cfg.HTTPPort)
	cmd := exec.Command("sudo", "sh", "-c",
		fmt.Sprintf(`echo '%s' | pfctl -ef - 2>/dev/null; sysctl -w net.inet.ip.forwarding=1 2>/dev/null; true`, rules))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func applyIptables(cfg *ProxyConfig) error {
	for _, rule := range [][]string{
		{"sudo", "iptables", "-t", "nat", "-A", "OUTPUT", "-p", "tcp", "--dport", "443", "-j", "REDIRECT", "--to-port", cfg.ProxyPort},
		{"sudo", "iptables", "-t", "nat", "-A", "OUTPUT", "-p", "tcp", "--dport", "80", "-j", "REDIRECT", "--to-port", cfg.HTTPPort},
	} {
		cmd := exec.Command(rule[0], rule[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

// port443InUse returns true if something is already listening on TCP port 443.
func port443InUse() bool {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:443", 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

