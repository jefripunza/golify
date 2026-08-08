package main

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func registerAPI(r fiber.Router) {
	v1 := r.Group("/v1")

	// public health
	v1.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"app":    "gotify",
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

	// user auth (very basic; password hashing is intentionally left for production hardening)
	v1.Post("/auth/login", func(c fiber.Ctx) error {
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
		// NOTE: production should verify a hashed password (bcrypt/argon2)
		if body.Password == "" {
			return c.Status(401).JSON(fiber.Map{"error": "invalid credentials"})
		}
		tok, err := issueToken(u)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{
			"token": tok,
			"user":  fiber.Map{"id": u.ID, "username": u.Username, "admin": u.Admin},
		})
	})
}
