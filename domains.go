package main

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// domainRegex matches a ROOT domain only: exactly two labels (label.label).
// Subdomains (3+ labels) are rejected — only root domains may be saved.
var domainRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?\.[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// DomainEntry is a standalone domain/subdomain entry in the global Domains
// list (sidebar menu "Domains"). Unlike Domain (which belongs to an
// Environment), these are free-standing records the user manages manually.
type DomainEntry struct {
	ID        UUID      `gorm:"primaryKey;size:36" json:"id"`
	Host      string    `gorm:"size:255;not null;uniqueIndex" json:"host"`
	CreatedAt time.Time `json:"created_at"`
}

// normalizeDomain strips scheme/path/query and lowercases, returns bare host.
// Accepts "example.com", "sub.example.com", "http://x.com", "https://sub.x.com/".
func normalizeDomain(raw string) (string, error) {
	d := strings.TrimSpace(raw)
	if d == "" {
		return "", errors.New("domain required")
	}
	// strip scheme if present
	if i := strings.Index(d, "://"); i >= 0 {
		d = d[i+3:]
	}
	// strip trailing path / query / fragment
	if i := strings.IndexAny(d, "/?#"); i >= 0 {
		d = d[:i]
	}
	d = strings.TrimSuffix(d, ".")
	d = strings.ToLower(d)
	if d == "" {
		return "", errors.New("domain required")
	}
	// structural validation: labels of [a-z0-9-], no leading/trailing '-',
	// at least one dot (domain or subdomain), no consecutive dots.
	if !domainRegex.MatchString(d) || strings.Contains(d, "..") {
		return "", errors.New("invalid domain format (example: example.com or sub.example.com)")
	}
	for _, label := range strings.Split(d, ".") {
		if label == "" || label[0] == '-' || label[len(label)-1] == '-' {
			return "", errors.New("invalid domain format (example: example.com or sub.example.com)")
		}
	}
	return d, nil
}

func registerDomains(r fiber.Router) {
	auth := r.Group("/domains", requireAuth)

	// list all
	auth.Get("/", func(c fiber.Ctx) error {
		var rows []DomainEntry
		if err := db.Order("id desc").Find(&rows).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(rows)
	})

	// create
	auth.Post("/", func(c fiber.Ctx) error {
		var body struct {
			Host string `json:"host"`
		}
		if err := c.Bind().JSON(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
		}
		host, err := normalizeDomain(body.Host)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		// server-side duplicate check (unique index would also catch)
		var count int64
		db.Model(&DomainEntry{}).Where("host = ?", host).Count(&count)
		if count > 0 {
			return c.Status(409).JSON(fiber.Map{"error": "domain already exists"})
		}
		row := DomainEntry{Host: host}
		if err := db.Create(&row).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(201).JSON(row)
	})

	// delete
	auth.Delete("/:id", func(c fiber.Ctx) error {
		id := c.Params("id")
		if err := db.Delete(&DomainEntry{}, "id = ?", id).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"deleted": id})
	})

	// duplicate check (used by FE before submit)
	auth.Post("/check", func(c fiber.Ctx) error {
		var body struct {
			Host string `json:"host"`
		}
		if err := c.Bind().JSON(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
		}
		host, err := normalizeDomain(body.Host)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		var count int64
		db.Model(&DomainEntry{}).Where("host = ?", host).Count(&count)
		return c.JSON(fiber.Map{"exists": count > 0})
	})

	_ = gorm.ErrRecordNotFound // keep gorm import used (pattern)
}
