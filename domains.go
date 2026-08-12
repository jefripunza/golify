package main

import (
	"errors"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v3"
)

// domainRegex matches a hostname: one or more labels, dot-separated.
// Root domains (example.com) and subdomains (sub.example.com) are both
// allowed. Final suffix is validated against domainSuffixes separately.
var domainRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$`)

// validDomainSuffix reports whether the final label(s) of host are a real
// domain suffix (TLD or multi-label public suffix like co.id). At least one
// label must precede the suffix.
func validDomainSuffix(host string) bool {
	labels := strings.Split(host, ".")
	if len(labels) < 2 {
		return false
	}
	// try the longest suffix first: full host minus first label, then
	// progressively shorter. The suffix must exist in domainSuffixes.
	for i := 1; i < len(labels); i++ {
		suffix := strings.Join(labels[i:], ".")
		if domainSuffixes[suffix] {
			return true
		}
	}
	return false
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
	// reject hostnames whose final suffix is not a real domain suffix
	// (e.g. "example.example", "foo.badexample") — TLD/public suffix must
	// exist in the IANA/PSL-backed domainSuffixes list.
	if !validDomainSuffix(d) {
		return "", errors.New("invalid domain suffix (must end with a valid TLD like .com, .id, .online, .co.id)")
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
		row := Domain{Host: host, EnvironmentID: body.EnvironmentID}
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
		row.EnvironmentID = body.EnvironmentID
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
