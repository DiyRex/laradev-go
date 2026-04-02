package proxy

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// StartDaemon launches the proxy as a detached background process (self proxy:daemon).
// Safe to call when already running — returns nil without doing anything.
func StartDaemon(cfg *ProxyConfig, projectDir string) error {
	if IsRunning(projectDir) {
		return nil
	}
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot find executable path: %w", err)
	}
	cmd := exec.Command(self, "proxy:daemon")
	cmd.Dir = projectDir
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start proxy daemon: %w", err)
	}
	return cmd.Process.Release()
}

// StopDaemon kills the running proxy daemon for the given project.
// Safe to call when not running — returns nil without doing anything.
func StopDaemon(projectDir string) error {
	pidFile := PIDFilePath(projectDir)
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return nil
	}
	var pid int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &pid); err != nil || pid <= 0 {
		os.Remove(pidFile)
		return nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		os.Remove(pidFile)
		return nil
	}
	_ = proc.Kill()
	os.Remove(pidFile)
	return nil
}

// RunDaemon starts the HTTPS reverse proxy for the given config.
// This is the entry point for the background daemon process started by proxy:up.
// It writes its PID to ~/.laradev/projects/{hash}/proxy.pid and blocks until killed.
func RunDaemon(cfg *ProxyConfig) error {
	if !cfg.IsConfigured() {
		return fmt.Errorf("proxy not configured — run: laradev proxy:setup")
	}

	// Write PID file so the TUI and CLI can detect we are running.
	pidFile := PIDFilePath(cfg.projectDir)
	if err := os.MkdirAll(filepath.Dir(pidFile), 0755); err != nil {
		return fmt.Errorf("cannot create pid dir: %w", err)
	}
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		return fmt.Errorf("cannot write pid file: %w", err)
	}
	defer os.Remove(pidFile)

	// Locate TLS certificate and key.
	certPath := filepath.Join(CertsDir(), cfg.Domain+".pem")
	keyPath := filepath.Join(CertsDir(), cfg.Domain+"-key.pem")

	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return fmt.Errorf("TLS certificate not found at %s\nRun: laradev proxy:setup", certPath)
	}

	tlsCert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return fmt.Errorf("failed to load TLS certificate: %w\nRun: laradev proxy:setup", err)
	}

	// Build the reverse proxy targeting the PHP server.
	target, err := url.Parse("http://127.0.0.1:" + cfg.TargetPort)
	if err != nil {
		return fmt.Errorf("invalid target port: %w", err)
	}
	rp := httputil.NewSingleHostReverseProxy(target)

	// Director: tell Laravel it's an HTTPS request on the canonical domain.
	// Without these headers Laravel generates redirect URLs with the wrong
	// scheme (http) and wrong port (8443).
	defaultDirector := rp.Director
	rp.Director = func(req *http.Request) {
		defaultDirector(req)
		// Strip any port from the Host header so Laravel sees just the domain.
		req.Host = cfg.Domain
		req.Header.Set("X-Forwarded-Proto", "https")
		req.Header.Set("X-Forwarded-Host", cfg.Domain)
		req.Header.Set("X-Forwarded-Port", "443")
		// REMOTE_ADDR hint for some middleware stacks.
		req.Header.Set("X-Real-IP", req.RemoteAddr)
	}

	rp.ModifyResponse = func(resp *http.Response) error {
		loc := resp.Header.Get("Location")
		if loc == "" {
			return nil
		}
		// Rewrite Location headers that use the wrong scheme or include :8443.
		// Handles redirects from the PHP server (localhost:PORT) and from
		// Laravel's own URL generation when it hasn't trusted the forwarded headers.
		for _, bad := range []string{
			"http://127.0.0.1:" + cfg.TargetPort,
			"http://0.0.0.0:" + cfg.TargetPort,
			"http://localhost:" + cfg.TargetPort,
			"http://" + cfg.Domain + ":" + cfg.ProxyPort,
			"https://" + cfg.Domain + ":" + cfg.ProxyPort,
			"http://" + cfg.Domain,
		} {
			if strings.HasPrefix(loc, bad) {
				newLoc := "https://" + cfg.Domain + loc[len(bad):]
				resp.Header.Set("Location", newLoc)
				return nil
			}
		}
		return nil
	}

	// HTTP server: redirect all traffic to HTTPS.
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		httpsURL := "https://" + cfg.Domain
		if cfg.ProxyPort != "443" {
			httpsURL += ":" + cfg.ProxyPort
		}
		httpsURL += r.RequestURI
		http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
	})
	httpServer := &http.Server{
		Addr:    "127.0.0.1:" + cfg.HTTPPort,
		Handler: httpMux,
	}
	go func() { _ = httpServer.ListenAndServe() }()

	// HTTPS server (blocking).
	httpsServer := &http.Server{
		Addr: "127.0.0.1:" + cfg.ProxyPort,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{tlsCert},
			MinVersion:   tls.VersionTLS12,
		},
		Handler: rp,
	}

	fmt.Printf("[laradev proxy] Listening on https://%s (→ localhost:%s)\n",
		cfg.AppURL(), cfg.TargetPort)

	return httpsServer.ListenAndServeTLS("", "")
}
