package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ensureSelfSignedCert generates a self-signed certificate for the given
// hostname (CN + SAN) into data/ssl/custom/<host>.crt / .key if it does not
// already exist. The TLS config's GetCertificate picks these up, so HTTPS
// works out of the box for any attached domain.
//
// Later this can be replaced by real ACME (Let's Encrypt) issuance — the
// loader already prefers data/ssl/letsencrypt/<host>/fullchain.pem when
// present.
func ensureSelfSignedCert(host string) error {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if host == "" {
		return fmt.Errorf("empty host")
	}
	crt := filepath.Join(dataDir, "ssl", "custom", host+".crt")
	key := filepath.Join(dataDir, "ssl", "custom", host+".key")
	if _, err := os.Stat(crt); err == nil {
		if _, err := os.Stat(key); err == nil {
			return nil // already generated
		}
	}
	if err := os.MkdirAll(filepath.Dir(crt), 0o755); err != nil {
		return err
	}
	// openssl req -x509 -newkey rsa:2048 -nodes -days 825 \
	//   -keyout <key> -out <crt> -subj "/CN=<host>" \
	//   -addext "subjectAltName=DNS:<host>,DNS:*.<host>"
	cmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes",
		"-days", "825",
		"-keyout", key, "-out", crt,
		"-subj", "/CN="+host,
		"-addext", "subjectAltName=DNS:"+host+",DNS:*."+host)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("openssl: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
