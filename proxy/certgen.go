package proxy

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// caDir holds the local CA key and certificate inside ~/.laradev/
func caDir() string { return filepath.Join(GlobalDir(), "ca") }
func caKeyFile() string  { return filepath.Join(caDir(), "ca.key") }
func caCertFile() string { return filepath.Join(caDir(), "ca.crt") }

// caTrustedFlag is written after the CA is trusted in the OS keychain,
// so we don't prompt the user on every proxy:setup run.
func caTrustedFlag() string { return filepath.Join(caDir(), ".trusted") }

// CAExists returns true if the local CA has been generated.
func CAExists() bool {
	_, keyErr := os.Stat(caKeyFile())
	_, certErr := os.Stat(caCertFile())
	return keyErr == nil && certErr == nil
}

// CAIsTrusted returns true if we've already added the CA to the OS trust store.
func CAIsTrusted() bool {
	_, err := os.Stat(caTrustedFlag())
	return err == nil
}

// EnsureCA creates the local CA if it doesn't exist, and trusts it in the OS
// keychain if not already trusted. Returns the loaded CA cert and key.
func EnsureCA() (*x509.Certificate, *ecdsa.PrivateKey, error) {
	if err := os.MkdirAll(caDir(), 0700); err != nil {
		return nil, nil, fmt.Errorf("create CA dir: %w", err)
	}

	if !CAExists() {
		fmt.Println("  Generating local CA…")
		if err := generateCA(); err != nil {
			return nil, nil, fmt.Errorf("generate CA: %w", err)
		}
		fmt.Println("  ✓ Local CA created")
	}

	ca, key, err := loadCA()
	if err != nil {
		return nil, nil, fmt.Errorf("load CA: %w", err)
	}

	if !CAIsTrusted() {
		fmt.Println("  Trusting CA in system keychain (may ask for your password)…")
		if err := trustCA(); err != nil {
			// Non-fatal: show a warning and continue. The cert will still work
			// in curl/Go; only browser display is affected.
			fmt.Printf("  ⚠  Could not auto-trust CA: %v\n", err)
			fmt.Printf("  ⚠  To trust manually run:\n")
			printManualTrustHint()
		} else {
			fmt.Println("  ✓ CA trusted in system keychain")
			// Mark as trusted so we don't ask again.
			_ = os.WriteFile(caTrustedFlag(), []byte("trusted"), 0644)
		}
	} else {
		fmt.Println("  ✓ CA already trusted")
	}

	return ca, key, nil
}

// GenerateDomainCert creates a TLS certificate for domain signed by the local CA
// and saves it to ~/.laradev/certs/<domain>.pem and <domain>-key.pem
func GenerateDomainCert(domain string) error {
	certPath := filepath.Join(CertsDir(), domain+".pem")
	keyPath := filepath.Join(CertsDir(), domain+"-key.pem")

	if err := os.MkdirAll(CertsDir(), 0755); err != nil {
		return err
	}

	// Regenerate if missing; skip if already present.
	if fi, err := os.Stat(certPath); err == nil && fi.Size() > 0 {
		fmt.Printf("  ✓ Certificate already exists: %s\n", certPath)
		return nil
	}

	ca, caKey, err := loadCA()
	if err != nil {
		return fmt.Errorf("load CA to sign domain cert: %w", err)
	}

	domainKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generate domain key: %w", err)
	}

	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: domain},
		DNSNames:     []string{domain},
		NotBefore:    time.Now().Add(-1 * time.Minute),
		NotAfter:     time.Now().Add(2 * 365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, ca, &domainKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("sign domain cert: %w", err)
	}

	keyBytes, err := x509.MarshalECPrivateKey(domainKey)
	if err != nil {
		return err
	}
	if err := writePEMFile(keyPath, "EC PRIVATE KEY", keyBytes, 0600); err != nil {
		return err
	}
	if err := writePEMFile(certPath, "CERTIFICATE", certDER, 0644); err != nil {
		return err
	}

	fmt.Printf("  ✓ Certificate generated: %s\n", certPath)
	return nil
}

// ── internal helpers ──────────────────────────────────────────────────────────

func generateCA() error {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "LaraDev Local CA",
			Organization: []string{"LaraDev"},
		},
		NotBefore:             time.Now().Add(-1 * time.Minute),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return err
	}

	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return err
	}
	if err := writePEMFile(caKeyFile(), "EC PRIVATE KEY", keyBytes, 0600); err != nil {
		return err
	}
	return writePEMFile(caCertFile(), "CERTIFICATE", certDER, 0644)
}

func loadCA() (*x509.Certificate, *ecdsa.PrivateKey, error) {
	certPEM, err := os.ReadFile(caCertFile())
	if err != nil {
		return nil, nil, err
	}
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, nil, fmt.Errorf("invalid CA cert PEM at %s", caCertFile())
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, err
	}

	keyPEM, err := os.ReadFile(caKeyFile())
	if err != nil {
		return nil, nil, err
	}
	block, _ = pem.Decode(keyPEM)
	if block == nil {
		return nil, nil, fmt.Errorf("invalid CA key PEM at %s", caKeyFile())
	}
	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}

// TrustCA forces the CA to be added to the OS trust store, removing any cached
// "already trusted" flag first. Call this from proxy:setup or proxy:trust.
func TrustCA() error {
	// Remove the flag so EnsureCA will re-trust even if it ran before.
	_ = os.Remove(caTrustedFlag())
	_, _, err := EnsureCA()
	return err
}

// trustCA adds the CA cert to the OS trust store.
func trustCA() error {
	switch runtime.GOOS {
	case "darwin":
		// Add to the System keychain (trusted by all browsers including Chrome/Safari).
		// Requires sudo — macOS will prompt for password.
		fmt.Println("  Adding CA to System keychain (sudo required for browser trust)…")
		cmd := exec.Command("sudo", "security", "add-trusted-cert",
			"-d", "-r", "trustRoot",
			"-k", "/Library/Keychains/System.keychain",
			caCertFile())
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		return cmd.Run()

	case "linux":
		dest := "/usr/local/share/ca-certificates/laradev-ca.crt"
		// Try direct write first (works when running as root or in CI).
		if err := copyFileContents(caCertFile(), dest); err == nil {
			return runCmd("update-ca-certificates")
		}
		// Fall back to sudo.
		cmd := exec.Command("sudo", "sh", "-c",
			fmt.Sprintf("cp '%s' '%s' && update-ca-certificates", caCertFile(), dest))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		return cmd.Run()
	}
	return nil
}

func printManualTrustHint() {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		fmt.Printf("    sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain %s\n", caCertFile())
		_ = home
	case "linux":
		fmt.Printf("    sudo cp %s /usr/local/share/ca-certificates/laradev-ca.crt\n", caCertFile())
		fmt.Println("    sudo update-ca-certificates")
	}
}

func writePEMFile(path, pemType string, der []byte, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: pemType, Bytes: der})
}

func copyFileContents(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
