package proxy

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

// GlobalDir returns ~/.laradev — all proxy state lives here, outside any project.
func GlobalDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/.laradev"
	}
	return filepath.Join(home, ".laradev")
}

// ProjectProxyDir returns ~/.laradev/projects/{16-char-hash}/
// The hash is derived from the absolute project path so each project gets its own slot.
func ProjectProxyDir(projectDir string) string {
	sum := sha256.Sum256([]byte(filepath.Clean(projectDir)))
	hash := fmt.Sprintf("%x", sum)[:16]
	return filepath.Join(GlobalDir(), "projects", hash)
}

// CertsDir returns ~/.laradev/certs/
func CertsDir() string {
	return filepath.Join(GlobalDir(), "certs")
}

// ProxyConfig holds per-project proxy settings stored globally (not in the project).
type ProxyConfig struct {
	Domain     string // e.g. "myapp.test"
	TargetPort string // PHP server port, e.g. "8007"
	ProxyPort  string // HTTPS listener port (no root needed), e.g. "8443"
	HTTPPort   string // HTTP redirect listener port, e.g. "8080"
	Enabled    bool

	projectDir string // set at load time, not written to disk
}

func defaultProxyConfig(projectDir, phpPort string) *ProxyConfig {
	return &ProxyConfig{
		Domain:     "",
		TargetPort: phpPort,
		ProxyPort:  "8443",
		HTTPPort:   "8080",
		Enabled:    false,
		projectDir: projectDir,
	}
}

// LoadProjectProxy reads proxy.conf from the global state dir for this project.
// Returns defaults (Enabled=false) if no config exists yet.
func LoadProjectProxy(projectDir, phpPort string) *ProxyConfig {
	cfg := defaultProxyConfig(projectDir, phpPort)
	path := filepath.Join(ProjectProxyDir(projectDir), "proxy.conf")

	f, err := os.Open(path)
	if err != nil {
		return cfg
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), "\"'")

		switch key {
		case "DOMAIN":
			cfg.Domain = val
		case "TARGET_PORT":
			cfg.TargetPort = val
		case "PROXY_PORT":
			cfg.ProxyPort = val
		case "HTTP_PORT":
			cfg.HTTPPort = val
		case "ENABLED":
			cfg.Enabled = val == "true"
		}
	}
	return cfg
}

// Save writes this config to ~/.laradev/projects/{hash}/proxy.conf
func (c *ProxyConfig) Save() error {
	dir := ProjectProxyDir(c.projectDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	enabled := "false"
	if c.Enabled {
		enabled = "true"
	}

	lines := []string{
		"# LaraDev Proxy Configuration — managed by laradev, do not edit manually",
		fmt.Sprintf("DOMAIN=\"%s\"", c.Domain),
		fmt.Sprintf("TARGET_PORT=\"%s\"", c.TargetPort),
		fmt.Sprintf("PROXY_PORT=\"%s\"", c.ProxyPort),
		fmt.Sprintf("HTTP_PORT=\"%s\"", c.HTTPPort),
		fmt.Sprintf("ENABLED=\"%s\"", enabled),
		"",
	}
	path := filepath.Join(dir, "proxy.conf")
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}

// IsConfigured returns true if a domain has been set and proxy is enabled.
func (c *ProxyConfig) IsConfigured() bool {
	return c.Domain != "" && c.Enabled
}

// AppURL returns the URL to show in the TUI — with port if != 443.
func (c *ProxyConfig) AppURL() string {
	if c.ProxyPort == "443" {
		return "https://" + c.Domain
	}
	return fmt.Sprintf("https://%s:%s", c.Domain, c.ProxyPort)
}

// PIDFilePath returns ~/.laradev/projects/{hash}/proxy.pid
func PIDFilePath(projectDir string) string {
	return filepath.Join(ProjectProxyDir(projectDir), "proxy.pid")
}

// IsRunning checks whether the proxy daemon is alive for this project.
func IsRunning(projectDir string) bool {
	data, err := os.ReadFile(PIDFilePath(projectDir))
	if err != nil {
		return false
	}
	var pid int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &pid); err != nil || pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0: no signal sent, but OS validates the PID exists and we can reach it.
	return proc.Signal(syscall.Signal(0)) == nil
}

// SlugifyDomain converts an app name to a .test domain slug.
// "My Laravel App" → "my-laravel-app.test"
// "LaraShop" → "larashop.test"
func SlugifyDomain(appName string) string {
	s := strings.ToLower(appName)

	// Replace spaces, underscores, dots with hyphens
	re := regexp.MustCompile(`[\s_.\\/]+`)
	s = re.ReplaceAllString(s, "-")

	// Remove any characters that are not alphanumeric or hyphen
	re2 := regexp.MustCompile(`[^a-z0-9-]`)
	s = re2.ReplaceAllString(s, "")

	// Collapse multiple hyphens
	re3 := regexp.MustCompile(`-{2,}`)
	s = re3.ReplaceAllString(s, "-")

	s = strings.Trim(s, "-")
	if s == "" {
		s = "laravel"
	}
	return s + ".test"
}

// EnsureGlobalDirs creates ~/.laradev/certs and ~/.laradev/projects if they don't exist.
func EnsureGlobalDirs() error {
	for _, dir := range []string{CertsDir(), filepath.Join(GlobalDir(), "projects")} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}
