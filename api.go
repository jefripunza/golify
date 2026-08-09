package main

import (
	"github.com/gofiber/fiber/v3"
)

func registerAPI(r fiber.Router) {
	v1 := r.Group("/v1")

	// public health
	v1.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"app":    "golify",
		})
	})

	// public messages list (read-only; full CRUD requires JWT below)
	v1.Get("/message", func(c fiber.Ctx) error {
		var rows []Message
		if err := db.Order("id desc").Limit(50).Find(&rows).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(rows)
	})

	// user auth (bcrypt-verified; MUST be declared BEFORE the `auth` group below —
	// Fiber v3 applies an empty-prefix group middleware to every route registered
	// after it, including /auth/login).
	v1.Post("/auth/login", loginHandler)

	// PaaS-style dashboard CRUD (JWT-protected)
	registerProjects(v1)

	// 7 other dashboard menus (servers, sources, s3, variables, keys, api-keys, mcp, teams)
	registerInfra(v1)

	// authenticated routes
	auth := v1.Group("", requireAuth)

	// create message
	auth.Post("/message", func(c fiber.Ctx) error {
		var body struct {
			Title    string `json:"title"`
			Message  string `json:"message"`
			Priority int    `json:"priority"`
		}
		if err := c.Bind().JSON(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		if body.Message == "" {
			return c.Status(400).JSON(fiber.Map{"error": "message required"})
		}
		m := Message{Title: body.Title, Message: body.Message, Priority: body.Priority}
		if err := db.Create(&m).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(201).JSON(m)
	})

	// delete one
	auth.Delete("/message/:id", func(c fiber.Ctx) error {
		id := c.Params("id")
		if err := db.Delete(&Message{}, id).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"deleted": id})
	})

	// admin-only: list + create applications + clients
	admin := v1.Group("/admin", requireAuth, requireAdmin)
	admin.Get("/application", func(c fiber.Ctx) error {
		var rows []Application
		if err := db.Find(&rows).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(rows)
	})
	admin.Post("/application", func(c fiber.Ctx) error {
		var body struct{ Name string `json:"name"` }
		if err := c.Bind().JSON(&body); err != nil || body.Name == "" {
			return c.Status(400).JSON(fiber.Map{"error": "name required"})
		}
		app := Application{Name: body.Name, Token: randomToken(16)}
		if err := db.Create(&app).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(201).JSON(app)
	})
	admin.Get("/client", func(c fiber.Ctx) error {
		var rows []Client
		if err := db.Find(&rows).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(rows)
	})

	// auth extras (JWT required)
	auth.Patch("/auth/password", changePassword)
	admin.Post("/auth/register", registerUser)
}
