package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ensureSelfSignedCert generates a self-signed certificate for the given
// hostname (CN + SAN) into data/ssl/letsencrypt/<host>/fullchain.pem and
// privkey.pem if it does not already exist — mirroring the layout a real
// ACME (Let's Encrypt) client would produce, so the TLS loader's existing
// letsencrypt preference handles it transparently.
//
// NOTE: data/ssl/custom/ is reserved for certificates the user purchases or
// uploads manually (Paid SSL); auto-generated certificates must never land
// there.
func ensureSelfSignedCert(host string) error {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if host == "" {
		return fmt.Errorf("empty host")
	}
	dir := filepath.Join(dataDir, "ssl", "letsencrypt", host)
	crt := filepath.Join(dir, "fullchain.pem")
	key := filepath.Join(dir, "privkey.pem")
	if _, err := os.Stat(crt); err == nil {
		if _, err := os.Stat(key); err == nil {
			return nil // already generated
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
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
