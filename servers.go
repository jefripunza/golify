package main

// servers.go — Server management: SSH connection test, system stats,
// secret encryption (AES-GCM) for SSH password / private key.
//
// Semua server dikelola dari satu dashboard Golify: register → test SSH →
// (opsional) join Kubernetes cluster → pantau stats. Secret TIDAK pernah
// dikirim balik ke FE (json:"-" di model).

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"golang.org/x/crypto/ssh"
)

// ─── Secret encryption (AES-GCM, key dari data/secret.key) ────────────────

var secretKey []byte

func initSecretKey() error {
	if secretKey != nil {
		return nil
	}
	kp := filepath.Join(dataDir, "secret.key")
	key, err := os.ReadFile(kp)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		key = make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return err
		}
		if err := os.WriteFile(kp, key, 0o600); err != nil {
			return err
		}
	}
	if len(key) != 32 {
		return errors.New("secret.key must be 32 bytes (AES-256)")
	}
	secretKey = key
	return nil
}

func encryptSecret(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	if err := initSecretKey(); err != nil {
		return "", err
	}
	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(ct), nil
}

func decryptSecret(enc string) (string, error) {
	if enc == "" {
		return "", nil
	}
	if err := initSecretKey(); err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce, ct := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

// ─── Server handlers (custom, bukan createGeneric) ─────────────────────────

// sanitizeServer zeroes secret fields before returning to FE.
func sanitizeServer(s *Server) {
	s.SSHPassword = ""
	s.SSHPrivateKey = ""
}

// ServerInput adalah DTO untuk binding JSON — tag json:"-" di Server model
// mencegah secret ter-bind, jadi kita bind ke struct ini dulu (field secret
// punya tag json normal), lalu salin ke Server.
type ServerInput struct {
	Name        string `json:"name"`
	Host        string `json:"host"`
	IP          string `json:"ip"`
	Region      string `json:"region"`
	Provider    string `json:"provider"`
	Status      string `json:"status"`
	CPU         float64 `json:"cpu"`
	Memory      int64   `json:"memory"`
	MemoryTotal int64   `json:"memory_total"`
	Disk        float64 `json:"disk"`
	Containers  int     `json:"containers"`
	KeyID       string  `json:"key_id"`

	SSHUser       string `json:"ssh_user"`
	SSHPort       int    `json:"ssh_port"`
	SSHAuthType   string `json:"ssh_auth_type"`
	SSHPassword   string `json:"ssh_password"`
	SSHPrivateKey string `json:"ssh_private_key"`
	SSHPublicKey  string `json:"ssh_public_key"`

	KubeEnabled     bool   `json:"kube_enabled"`
	KubeRole        string `json:"kube_role"`
	KubeVersion     string `json:"kube_version"`
	KubeJoinCommand string `json:"kube_join_command"`
	KubeCluster     string `json:"kube_cluster"`

	Labels string `json:"labels"`
	Notes  string `json:"notes"`
}

func serverFromInput(in *ServerInput) Server {
	return Server{
		Name:            in.Name,
		Host:            in.Host,
		IP:              in.IP,
		Region:          in.Region,
		Provider:        in.Provider,
		Status:          in.Status,
		CPU:             in.CPU,
		Memory:          in.Memory,
		MemoryTotal:     in.MemoryTotal,
		Disk:            in.Disk,
		Containers:      in.Containers,
		KeyID:           UUID(in.KeyID),
		SSHUser:         in.SSHUser,
		SSHPort:         in.SSHPort,
		SSHAuthType:     in.SSHAuthType,
		SSHPassword:     in.SSHPassword,
		SSHPrivateKey:   in.SSHPrivateKey,
		SSHPublicKey:    in.SSHPublicKey,
		KubeEnabled:     in.KubeEnabled,
		KubeRole:        in.KubeRole,
		KubeVersion:     in.KubeVersion,
		KubeJoinCommand: in.KubeJoinCommand,
		KubeCluster:     in.KubeCluster,
		Labels:          in.Labels,
		Notes:           in.Notes,
	}
}

func registerServer(ctx fiber.Ctx) error {
	if err := initSecretKey(); err != nil {
		return ctx.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	var in ServerInput
	if err := ctx.Bind().JSON(&in); err != nil {
		return ctx.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	s := serverFromInput(&in)
	// Sanitasi input: host/name wajib.
	if s.Name == "" || s.Host == "" {
		return ctx.Status(400).JSON(fiber.Map{"error": "name and host are required"})
	}
	if s.SSHPort == 0 {
		s.SSHPort = 22
	}
	if s.SSHUser == "" {
		s.SSHUser = "root"
	}
	if s.SSHAuthType == "" {
		s.SSHAuthType = "password"
	}
	// Enkripsi secret sebelum simpan.
	encPass, err := encryptSecret(s.SSHPassword)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"error": "encrypt password: " + err.Error()})
	}
	encKey, err := encryptSecret(s.SSHPrivateKey)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"error": "encrypt key: " + err.Error()})
	}
	s.SSHPassword = encPass
	s.SSHPrivateKey = encKey
	if err := db.Create(&s).Error; err != nil {
		return ctx.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	sanitizeServer(&s)
	return ctx.Status(201).JSON(s)
}

func updateServer(ctx fiber.Ctx) error {
	if err := initSecretKey(); err != nil {
		return ctx.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	id := ctx.Params("id")
	var existing Server
	if err := db.First(&existing, "id = ?", id).Error; err != nil {
		return ctx.Status(404).JSON(fiber.Map{"error": "not found"})
	}
	var in ServerInput
	if err := ctx.Bind().JSON(&in); err != nil {
		return ctx.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	s := serverFromInput(&in)
	s.ID = existing.ID
	// Secret kosong dari FE = pertahankan yang lama.
	// Secret terisi = enkripsi baru.
	if s.SSHPassword == "" {
		s.SSHPassword = existing.SSHPassword
	} else {
		enc, err := encryptSecret(s.SSHPassword)
		if err != nil {
			return ctx.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		s.SSHPassword = enc
	}
	if s.SSHPrivateKey == "" {
		s.SSHPrivateKey = existing.SSHPrivateKey
	} else {
		enc, err := encryptSecret(s.SSHPrivateKey)
		if err != nil {
			return ctx.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		s.SSHPrivateKey = enc
	}
	if err := db.Save(&s).Error; err != nil {
		return ctx.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	sanitizeServer(&s)
	return ctx.JSON(s)
}

// ─── SSH client ────────────────────────────────────────────────────────────

// dialServer builds an ssh.ClientConfig from stored (decrypted) secrets.
func dialServer(s *Server) (*ssh.Client, error) {
	if err := initSecretKey(); err != nil {
		return nil, err
	}
	port := s.SSHPort
	if port == 0 {
		port = 22
	}
	user := s.SSHUser
	if user == "" {
		user = "root"
	}
	addr := net.JoinHostPort(s.Host, strconv.Itoa(port))

	var auth []ssh.AuthMethod
	switch s.SSHAuthType {
	case "private_key":
		keyPEM, err := decryptSecret(s.SSHPrivateKey)
		if err != nil || strings.TrimSpace(keyPEM) == "" {
			return nil, errors.New("private key tidak tersedia / gagal decrypt")
		}
		signer, err := ssh.ParsePrivateKey([]byte(keyPEM))
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
		auth = append(auth, ssh.PublicKeys(signer))
	default: // password
		pass, err := decryptSecret(s.SSHPassword)
		if err != nil || pass == "" {
			return nil, errors.New("password tidak tersedia / gagal decrypt")
		}
		auth = append(auth, ssh.Password(pass))
	}

	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: host key pinning
		Timeout:         8 * time.Second,
	}
	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// runSSH executes a command and returns stdout (+stderr merged).
func runSSH(client *ssh.Client, cmd string) (string, error) {
	sess, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer sess.Close()
	out, err := sess.CombinedOutput(cmd)
	if err != nil {
		return string(out), err
	}
	return string(out), nil
}

// ─── POST /servers/:id/test — koneksi SSH nyata ────────────────────────────

func testServerSSH(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	var s Server
	if err := db.First(&s, "id = ?", id).Error; err != nil {
		return ctx.Status(404).JSON(fiber.Map{"error": "not found"})
	}
	client, err := dialServer(&s)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{"ok": false, "error": err.Error()})
	}
	defer client.Close()

	// Kumpulkan info sistem sekaligus (satu koneksi).
	info := map[string]any{}
	out, err := runSSH(client, "uname -a 2>/dev/null | head -1; echo '---'; cat /etc/os-release 2>/dev/null | grep '^PRETTY_NAME=' | cut -d= -f2; echo '---'; hostname; echo '---'; uptime -p 2>/dev/null")
	if err == nil {
		parts := strings.SplitN(out, "---", 4)
		if len(parts) == 4 {
			info["uname"] = strings.TrimSpace(parts[0])
			info["os"] = strings.Trim(strings.TrimSpace(parts[1]), `"`)
			info["hostname"] = strings.TrimSpace(parts[2])
			info["uptime"] = strings.TrimSpace(parts[3])
		}
	}

	// Update status jadi online + metadata.
	s.Status = "online"
	db.Model(&s).Updates(map[string]any{"status": "online", "host": s.Host})
	return ctx.JSON(fiber.Map{"ok": true, "info": info})
}

// ─── GET /servers/:id/stats — statistik sistem real via SSH ───────────────

func serverStats(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	var s Server
	if err := db.First(&s, "id = ?", id).Error; err != nil {
		return ctx.Status(404).JSON(fiber.Map{"error": "not found"})
	}
	client, err := dialServer(&s)
	if err != nil {
		return ctx.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	defer client.Close()

	stats := map[string]any{}

	// CPU: 1 - idle/total dari /proc/stat
	if out, err := runSSH(client, "grep '^cpu ' /proc/stat | awk '{print $2+$3+$4+$5+$6+$7+$8, $5}'"); err == nil {
		var total, idle float64
		if _, err := fmt.Sscanf(strings.TrimSpace(out), "%f %f", &total, &idle); err == nil && total > 0 {
			stats["cpu"] = round1((1 - idle/total) * 100)
		}
	}
	// Memory: /proc/meminfo
	if out, err := runSSH(client, "grep -E '^(MemTotal|MemAvailable):' /proc/meminfo | awk '{print $2}'"); err == nil {
		lines := strings.Fields(strings.TrimSpace(out))
		if len(lines) == 2 {
			totalKB, _ := strconv.ParseFloat(lines[0], 64)
			availKB, _ := strconv.ParseFloat(lines[1], 64)
			usedMB := (totalKB - availKB) / 1024
			stats["memory_used_mb"] = int64(usedMB)
			stats["memory_total_mb"] = int64(totalKB / 1024)
		}
	}
	// Disk: df /
	if out, err := runSSH(client, "df -P / | awk 'NR==2 {print $5}' | tr -d '%'"); err == nil {
		if pct, err := strconv.ParseFloat(strings.TrimSpace(out), 64); err == nil {
			stats["disk_pct"] = round1(pct)
		}
	}
	// Containers: podman/docker
	if out, err := runSSH(client, "(podman ps -q 2>/dev/null | wc -l) || (docker ps -q 2>/dev/null | wc -l) || echo 0"); err == nil {
		if n, err := strconv.Atoi(strings.TrimSpace(out)); err == nil {
			stats["containers"] = n
		}
	}
	// Kubernetes: kubectl node status
	if out, err := runSSH(client, "kubectl get nodes -o name 2>/dev/null | wc -l"); err == nil {
		if n, err := strconv.Atoi(strings.TrimSpace(out)); err == nil {
			stats["kube_nodes"] = n
		}
	}
	// Uptime seconds
	if out, err := runSSH(client, "awk '{print int($1)}' /proc/uptime"); err == nil {
		if sec, err := strconv.ParseFloat(strings.TrimSpace(out), 64); err == nil {
			stats["uptime_sec"] = int64(sec)
		}
	}
	// OS string
	if out, err := runSSH(client, "cat /etc/os-release 2>/dev/null | grep '^PRETTY_NAME=' | cut -d= -f2 | tr -d '\"'"); err == nil {
		stats["os"] = strings.TrimSpace(out)
	}
	// Kernel
	if out, err := runSSH(client, "uname -r"); err == nil {
		stats["kernel"] = strings.TrimSpace(out)
	}
	// IP public (best effort)
	if out, err := runSSH(client, "hostname -I 2>/dev/null | awk '{print $1}'"); err == nil && strings.TrimSpace(out) != "" {
		stats["ip"] = strings.TrimSpace(out)
	}

	// Persist ke DB agar dashboard tampil real.
	db.Model(&s).Updates(map[string]any{
		"status":       "online",
		"cpu":          stats["cpu"],
		"memory":       stats["memory_used_mb"],
		"memory_total": stats["memory_total_mb"],
		"disk":         stats["disk_pct"],
		"containers":   stats["containers"],
	})
	return ctx.JSON(stats)
}

func round1(v float64) float64 { return float64(int(v*10)) / 10 }

// guard: log import dipakai oleh handler lain; biarkan saja.
var _ = log.Printf
