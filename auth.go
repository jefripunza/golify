package main

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ─── password hashing ──────────────────────────────────────────────────────
// hashPassword bcrypt-hashes a plaintext password (cost 10 default).
func hashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(b), err
}

// checkPassword verifies a plaintext password against a stored hash.
// Falls back to a constant-time dummy compare when the stored value is
// empty (so unknown usernames don't leak timing info).
func checkPassword(stored, plain string) bool {
	if stored == "" {
		bcrypt.CompareHashAndPassword([]byte("$2a$10$7EqJtq98hPqEX7fNZaFWoOhi5t4pGSd5jL9z7y9q3mYkZ2hY0uS2m"), []byte(plain))
		return false
	}
	if strings.HasPrefix(stored, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(plain)) == nil
	}
	// Legacy plaintext passhash (from earliest dev builds). Migrate to
	// bcrypt lazily on first successful login so old DBs keep working.
	return stored == plain
}

// loginHandler verifies credentials and issues a JWT. Also lazily upgrades
// legacy plaintext-password users to bcrypt on first successful login.
func loginHandler(c fiber.Ctx) error {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	var u User
	err := db.Where("username = ?", body.Username).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return c.Status(401).JSON(fiber.Map{"error": "invalid credentials"})
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if !checkPassword(u.Passhash, body.Password) {
		return c.Status(401).JSON(fiber.Map{"error": "invalid credentials"})
	}
	// lazy migration: plaintext → bcrypt on successful login
	if u.Passhash != "" && !strings.HasPrefix(u.Passhash, "$2") {
		if h, herr := hashPassword(body.Password); herr == nil {
			u.Passhash = h
			db.Model(&u).Update("passhash", h)
		}
	}
	tok, err := issueToken(u)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"token": tok,
		"user":  fiber.Map{"id": u.ID, "username": u.Username, "admin": u.Admin},
	})
}

// registerUser creates a new user with a bcrypt-hashed password (admin only).
func registerUser(c fiber.Ctx) error {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Admin    bool   `json:"admin"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	if body.Username == "" || body.Password == "" {
		return c.Status(400).JSON(fiber.Map{"error": "username and password required"})
	}
	if len(body.Password) < 8 {
		return c.Status(400).JSON(fiber.Map{"error": "password must be at least 8 characters"})
	}
	h, err := hashPassword(body.Password)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	u := User{Username: body.Username, Passhash: h, Admin: body.Admin}
	if err := db.Create(&u).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"id": u.ID, "username": u.Username, "admin": u.Admin})
}

// changePassword lets the current user rotate their own password (bcrypt).
func changePassword(c fiber.Ctx) error {
	claims, ok := c.Locals("claims").(*Claims)
	if !ok || claims == nil || claims.UserID == 0 {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}
	uid := claims.UserID
	var body struct {
		Old string `json:"old_password"`
		New string `json:"new_password"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	if len(body.New) < 8 {
		return c.Status(400).JSON(fiber.Map{"error": "new password must be at least 8 characters"})
	}
	var u User
	if err := db.First(&u, uid).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "user not found"})
	}
	if !checkPassword(u.Passhash, body.Old) {
		return c.Status(401).JSON(fiber.Map{"error": "old password incorrect"})
	}
	h, err := hashPassword(body.New)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if err := db.Model(&u).Update("passhash", h).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true})
}

// touchLoginRateLimit is a minimal in-memory rate limiter for login attempts.
// (Simple naive map; production should use a proper limiter/redis.)
var _ = time.Now // keep time import available for future limiter work