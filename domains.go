package main

import (
	"errors"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v3"
)

// domainRegex matches a hostname: one or more labels, dot-separated.
// Only root domains are accepted (see normalizeDomain below).
var domainRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$`)

// validDomainSuffix reports whether the final label(s) of host are a real
// domain suffix (TLD or multi-label public suffix like co.id). At least one
// label must precede the suffix — a bare suffix (co.id, com) is NOT a valid
// domain. Tries the longest suffix first (full host down to the TLD).
func validDomainSuffix(host string) bool {
	labels := strings.Split(host, ".")
	if len(labels) < 2 {
		return false
	}
	for i := 0; i < len(labels); i++ {
		suffix := strings.Join(labels[i:], ".")
		if domainSuffixes[suffix] {
			return i > 0 // at least one label before the suffix
		}
	}
	return false
}

// normalizeDomain strips scheme/path/query, lowercases, strips a leading
// "www." label (www.example.com ≡ example.com), returns bare host.
// Accepts only ROOT domains — e.g. "example.com" — and REJECTS subdomains
// (sub.example.com), per Pak Jefri's hard rule: no subdomains, TLD must be
// valid. Bare suffixes (co.id) are also rejected.
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
	// www.example.com ≡ example.com — drop the www. label (but only the
	// exact "www" label, not "wwws.example.com").
	d = strings.TrimPrefix(d, "www.")
	if d == "" {
		return "", errors.New("domain required")
	}
	// structural validation: labels of [a-z0-9-], no leading/trailing '-',
	// at least one dot, no consecutive dots.
	if !domainRegex.MatchString(d) || strings.Contains(d, "..") {
		return "", errors.New("invalid domain format (example: example.com)")
	}
	for _, label := range strings.Split(d, ".") {
		if label == "" || label[0] == '-' || label[len(label)-1] == '-' {
			return "", errors.New("invalid domain format (example: example.com)")
		}
	}
	// reject hostnames whose final suffix is not a real domain suffix
	// (e.g. "example.example", "foo.badexample") — TLD/public suffix must
	// exist in the IANA/PSL-backed domainSuffixes list.
	if !validDomainSuffix(d) {
		return "", errors.New("invalid domain suffix (must end with a valid TLD like .com, .id, .online, .co.id)")
	}
	// HARD RULE (Pak Jefri): only ROOT domains are allowed — the host
	// must be exactly ONE label + the longest matching domain suffix.
	// Examples: example.com (1+1), foo.co.id (1 + co.id public suffix).
	// Subdomains like sub.example.com or golify.sawang.tech are REJECTED.
	labels := strings.Split(d, ".")
	suffixLen := 0
	for i := 0; i < len(labels); i++ {
		if domainSuffixes[strings.Join(labels[i:], ".")] {
			suffixLen = len(labels) - i
			break
		}
	}
	if suffixLen == 0 || len(labels)-suffixLen != 1 {
		return "", errors.New("subdomains are not allowed — use a root domain only (example: example.com)")
	}
	return d, nil
}

func registerDomains(r fiber.Router) {
	auth := r.Group("/domains", requireAuth)

	// list all (domains table — single source of truth)
	auth.Get("/", func(c fiber.Ctx) error {
		var rows []Domain
		if err := db.Order("id desc").Find(&rows).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(rows)
	})

	// create
	auth.Post("/", func(c fiber.Ctx) error {
		var body struct {
			Host          string `json:"host"`
			EnvironmentID string `json:"environment_id"`
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
		db.Model(&Domain{}).Where("host = ?", host).Count(&count)
		if count > 0 {
			return c.Status(409).JSON(fiber.Map{"error": "domain already exists"})
		}
		row := Domain{Host: host}
		if body.EnvironmentID != "" {
			envID := UUID(body.EnvironmentID)
			row.EnvironmentID = &envID
		}
		if err := db.Create(&row).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(201).JSON(row)
	})

	// update (edit host / link environment)
	auth.Patch("/:id", func(c fiber.Ctx) error {
		id := c.Params("id")
		var body struct {
			Host          string `json:"host"`
			EnvironmentID string `json:"environment_id"`
		}
		if err := c.Bind().JSON(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
		}
		host, err := normalizeDomain(body.Host)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		var row Domain
		if err := db.First(&row, "id = ?", id).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "domain not found"})
		}
		// duplicate check excluding self
		var count int64
		db.Model(&Domain{}).Where("host = ? AND id <> ?", host, id).Count(&count)
		if count > 0 {
			return c.Status(409).JSON(fiber.Map{"error": "domain already exists"})
		}
		row.Host = host
		if body.EnvironmentID == "" {
			row.EnvironmentID = nil
		} else {
			envID := UUID(body.EnvironmentID)
			row.EnvironmentID = &envID
		}
		if err := db.Save(&row).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(row)
	})

	// delete
	auth.Delete("/:id", func(c fiber.Ctx) error {
		id := c.Params("id")
		if err := db.Delete(&Domain{}, "id = ?", id).Error; err != nil {
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
		db.Model(&Domain{}).Where("host = ?", host).Count(&count)
		return c.JSON(fiber.Map{"exists": count > 0})
	})
}
