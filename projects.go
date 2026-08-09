package main

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// registerProjects wires the PaaS-style project/env/service CRUD under /api/v1.
// All routes require a JWT (requireAuth) unless noted.
func registerProjects(r fiber.Router) {
	auth := r.Group("/projects", requireAuth)

	// ─── Projects ──────────────────────────────────────────────────────────
	auth.Get("/", func(c fiber.Ctx) error {
		var rows []Project
		if err := db.Preload("Envs").Order("id desc").Find(&rows).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(rows)
	})

	auth.Post("/", func(c fiber.Ctx) error {
		var body struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			SourceID    string `json:"source_id"`
		}
		if err := c.Bind().JSON(&body); err != nil || body.Name == "" {
			return c.Status(400).JSON(fiber.Map{"error": "name required"})
		}
		p := Project{Name: body.Name, Description: body.Description, SourceID: body.SourceID}
		if err := db.Create(&p).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(201).JSON(p)
	})

	auth.Get("/:id", func(c fiber.Ctx) error {
		var p Project
		err := db.Preload("Envs").Preload("Envs.Services").Preload("Envs.Domains").
			First(&p, "id = ?", c.Params("id")).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": "project not found"})
		}
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(p)
	})

	auth.Delete("/:id", func(c fiber.Ctx) error {
		if err := db.Delete(&Project{}, c.Params("id")).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"deleted": c.Params("id")})
	})

	// ─── Environments within a project ─────────────────────────────────────
	auth.Get("/:projectId/environments", func(c fiber.Ctx) error {
		var rows []Environment
		if err := db.Where("project_id = ?", c.Params("projectId")).
			Preload("Services").Preload("Domains").Find(&rows).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(rows)
	})

	auth.Post("/:projectId/environments", func(c fiber.Ctx) error {
		var body struct {
			Name         string   `json:"name"`
			IsProduction bool     `json:"is_production"`
			Domains      []string `json:"domains"`
		}
		if err := c.Bind().JSON(&body); err != nil || body.Name == "" {
			return c.Status(400).JSON(fiber.Map{"error": "name required"})
		}
		pid, err := strconv.Atoi(c.Params("projectId"))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "bad project id"})
		}
		env := Environment{ProjectID: uint(pid), Name: body.Name, IsProduction: body.IsProduction}
		for _, h := range body.Domains {
			env.Domains = append(env.Domains, Domain{Host: h})
		}
		if err := db.Create(&env).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(201).JSON(env)
	})

	// ─── Services within an environment ────────────────────────────────────
	auth.Get("/:projectId/environments/:envId/services", func(c fiber.Ctx) error {
		var rows []Service
		if err := db.Where("environment_id = ?", c.Params("envId")).Find(&rows).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(rows)
	})

	auth.Post("/:projectId/environments/:envId/services", func(c fiber.Ctx) error {
		var body struct {
			Name        string   `json:"name"`
			Kind        string   `json:"kind"`
			Image       string   `json:"image"`
			ComposePath string   `json:"compose_path"`
			Status      string   `json:"status"`
			CPU         float64  `json:"cpu"`
			Memory      int64    `json:"memory"`
			Ports       []string `json:"ports"`
		}
		if err := c.Bind().JSON(&body); err != nil || body.Name == "" {
			return c.Status(400).JSON(fiber.Map{"error": "name required"})
		}
		eid, err := strconv.Atoi(c.Params("envId"))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "bad env id"})
		}
		if body.Kind == "" {
			body.Kind = "container"
		}
		if body.Status == "" {
			body.Status = "stopped"
		}
		s := Service{
			EnvironmentID: uint(eid), Name: body.Name, Kind: body.Kind,
			Image: body.Image, ComposePath: body.ComposePath, Status: body.Status,
			CPU: body.CPU, Memory: body.Memory, Ports: body.Ports,
		}
		if err := db.Create(&s).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(201).JSON(s)
	})

	// ─── Start / Stop / Restart a service (status switch) ─────────────────
	auth.Post("/:projectId/environments/:envId/services/:serviceId/start", func(c fiber.Ctx) error {
		return setServiceStatus(c, "running")
	})
	auth.Post("/:projectId/environments/:envId/services/:serviceId/stop", func(c fiber.Ctx) error {
		return setServiceStatus(c, "stopped")
	})

	// fallback: direct /api/v1/services?env=<id> convenience
	r.Get("/services", requireAuth, func(c fiber.Ctx) error {
		var rows []Service
		q := db
		if env := c.Query("env"); env != "" {
			q = q.Where("environment_id = ?", env)
		}
		if err := q.Order("id desc").Limit(200).Find(&rows).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(rows)
	})
}

func setServiceStatus(c fiber.Ctx, status string) error {
	sid, err := strconv.Atoi(c.Params("serviceId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "bad service id"})
	}
	var s Service
	if err := db.First(&s, sid).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "service not found"})
	}
	s.Status = status
	if err := db.Save(&s).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(s)
}