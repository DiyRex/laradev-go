package proxy

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// SetupProxy performs the one-time setup for a project's HTTPS proxy:
//  1. Ensures ~/.laradev dirs exist
//  2. Checks mkcert is installed (prints install hint if not)
//  3. Generates a trusted TLS certificate for the domain
//  4. Adds the domain to /etc/hosts (127.0.0.1 <domain>)
//  5. Saves the proxy.conf to ~/.laradev/projects/{hash}/
//
// Port forwarding (443 → ProxyPort) is NOT done automatically because it
// always requires root. Instructions are printed instead so users can decide.
func SetupProxy(cfg *ProxyConfig) error {
	fmt.Println("  Setting up HTTPS proxy for " + cfg.Domain + "…")
	fmt.Println()

	// 1. Global dirs
	if err := EnsureGlobalDirs(); err != nil {
		return fmt.Errorf("cannot create ~/.laradev dirs: %w", err)
	}

	// 2. Check mkcert
	if err := CheckMkcert(); err != nil {
		return err
	}

	// 3. Generate certificate
	if err := GenerateCert(cfg.Domain); err != nil {
		return err
	}

	// 4. /etc/hosts entry
	if err := AddHostsEntry(cfg.Domain); err != nil {
		return fmt.Errorf("failed to update /etc/hosts: %w", err)
	}

	// 5. Save config
	cfg.Enabled = true
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("cannot save proxy config: %w", err)
	}

	fmt.Println()
	fmt.Println("  ✓ Done! Start the proxy with: laradev proxy:up")
	fmt.Printf("  ✓ Your app will be at: %s\n", cfg.AppURL())
	fmt.Println()
	printPortForwardingHint(cfg)

	return nil
}

// CheckMkcert verifies mkcert is installed and the local CA is trusted.
func CheckMkcert() error {
	if _, err := exec.LookPath("mkcert"); err != nil {
		fmt.Println("  ✗ mkcert not found.")
		printMkcertInstallHint()
		return fmt.Errorf("mkcert is required — install it then re-run: laradev proxy:setup")
	}

	fmt.Println("  ✓ mkcert found")

	// Install the local CA into the system trust store (idempotent — safe to run again).
	cmd := exec.Command("mkcert", "-install")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mkcert -install failed: %w", err)
	}
	fmt.Println("  ✓ local CA trusted")
	return nil
}

// GenerateCert creates a mkcert-signed certificate for domain and stores it
// in ~/.laradev/certs/<domain>.pem and ~/.laradev/certs/<domain>-key.pem
func GenerateCert(domain string) error {
	certPath := filepath.Join(CertsDir(), domain+".pem")
	keyPath := filepath.Join(CertsDir(), domain+"-key.pem")

	// Skip if already exists and is non-empty.
	if fi, err := os.Stat(certPath); err == nil && fi.Size() > 0 {
		fmt.Printf("  ✓ certificate already exists: %s\n", certPath)
		return nil
	}

	cmd := exec.Command("mkcert",
		"-cert-file", certPath,
		"-key-file", keyPath,
		domain,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mkcert cert generation failed: %w", err)
	}
	fmt.Printf("  ✓ certificate generated: %s\n", certPath)
	return nil
}

// AddHostsEntry appends "127.0.0.1 <domain> # laradev" to /etc/hosts if not present.
// Requires write access to /etc/hosts — prompts sudo if needed.
func AddHostsEntry(domain string) error {
	hostsFile := "/etc/hosts"

	// Check if the entry already exists.
	data, err := os.ReadFile(hostsFile)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", hostsFile, err)
	}
	if strings.Contains(string(data), domain) {
		fmt.Printf("  ✓ /etc/hosts already has entry for %s\n", domain)
		return nil
	}

	entry := fmt.Sprintf("\n127.0.0.1 %s # laradev\n", domain)

	// Try direct write first (works if running as root or file is world-writable).
	f, err := os.OpenFile(hostsFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()
		if _, werr := f.WriteString(entry); werr != nil {
			return werr
		}
		fmt.Printf("  ✓ added to /etc/hosts: 127.0.0.1 %s\n", domain)
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
	fmt.Printf("  ✓ added to /etc/hosts: 127.0.0.1 %s\n", domain)
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

	// Try direct write.
	if err := os.WriteFile(hostsFile, []byte(cleaned), 0644); err != nil {
		// Sudo fallback.
		cmd := exec.Command("sudo", "sh", "-c",
			fmt.Sprintf(`grep -v "%s.*laradev" /etc/hosts > /tmp/.laradev_hosts && sudo mv /tmp/.laradev_hosts /etc/hosts`, domain))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		return cmd.Run()
	}
	return nil
}

// SetupPortForwarding prints instructions (and optionally applies) OS-level port
// redirection so that port 443 → ProxyPort and 80 → HTTPPort.
// This is called after a successful proxy:setup to inform the user.
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

// printPortForwardingHint prints OS-specific instructions to enable port 443→8443 redirect.
func printPortForwardingHint(cfg *ProxyConfig) {
	fmt.Println("  Optional: enable port forwarding so https works without :8443")
	fmt.Println()
	switch runtime.GOOS {
	case "darwin":
		fmt.Println("  Run once (resets on reboot):")
		fmt.Printf("    sudo sh -c \"echo 'rdr pass on lo0 proto tcp from any to any port 443 -> 127.0.0.1 port %s' | pfctl -ef -\"\n", cfg.ProxyPort)
		fmt.Println()
		fmt.Println("  To auto-apply on boot, run: laradev proxy:ports")
	case "linux":
		fmt.Println("  Run once (resets on reboot):")
		fmt.Printf("    sudo iptables -t nat -A OUTPUT -p tcp --dport 443 -j REDIRECT --to-port %s\n", cfg.ProxyPort)
		fmt.Println()
		fmt.Println("  To persist across reboots, save iptables rules with your distro's method.")
		fmt.Println("  To auto-apply on boot, run: laradev proxy:ports")
	default:
		fmt.Printf("  Forward port 443 → %s using your system's firewall.\n", cfg.ProxyPort)
	}
	fmt.Println()
}

func printMkcertInstallHint() {
	fmt.Println()
	fmt.Println("  Install mkcert:")
	fmt.Println("    macOS:  brew install mkcert")
	fmt.Println("    Linux (apt):  sudo apt install mkcert")
	fmt.Println("    Linux (manual): https://github.com/FiloSottile/mkcert/releases")
	fmt.Println()
}
