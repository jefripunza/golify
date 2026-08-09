package main

import (
	"github.com/gofiber/fiber/v3"
)

// registerInfra wires CRUD for the 7 remaining dashboard menus.
// All routes are JWT-protected (requireAuth applied by caller group).
// Each menu gets: GET /<name>  (list), POST /<name> (create),
//                GET/PATCH/DELETE /<name>/:id (read/update/delete).
func registerInfra(r fiber.Router) {
	// name → model factory. JSON bodies are decoded generically, then a
	// per-model struct is built with the same field names (snake_case JSON).
	c := r.Group("/")

	c.Get("/servers", func(ctx fiber.Ctx) error {
		var rows []Server
		if err := db.Order("id desc").Find(&rows).Error; err != nil {
			return ctx.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.JSON(rows)
	})
	c.Post("/servers", func(ctx fiber.Ctx) error { return createGeneric(ctx, &Server{}) })
	c.Get("/servers/:id", func(ctx fiber.Ctx) error { return getGeneric(ctx, &Server{}) })
	c.Patch("/servers/:id", func(ctx fiber.Ctx) error { return patchGeneric(ctx, &Server{}) })
	c.Delete("/servers/:id", func(ctx fiber.Ctx) error { return deleteGeneric(ctx, &Server{}) })

	c.Get("/sources", func(ctx fiber.Ctx) error {
		var rows []Source
		if err := db.Order("id desc").Find(&rows).Error; err != nil {
			return ctx.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.JSON(rows)
	})
	c.Post("/sources", func(ctx fiber.Ctx) error { return createGeneric(ctx, &Source{}) })
	c.Delete("/sources/:id", func(ctx fiber.Ctx) error { return deleteGeneric(ctx, &Source{}) })

	c.Get("/s3", func(ctx fiber.Ctx) error {
		var rows []S3Storage
		if err := db.Order("id desc").Find(&rows).Error; err != nil {
			return ctx.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.JSON(rows)
	})
	c.Post("/s3", func(ctx fiber.Ctx) error { return createGeneric(ctx, &S3Storage{}) })
	c.Delete("/s3/:id", func(ctx fiber.Ctx) error { return deleteGeneric(ctx, &S3Storage{}) })

	c.Get("/variables", func(ctx fiber.Ctx) error {
		var rows []SharedVariable
		if err := db.Order("id desc").Find(&rows).Error; err != nil {
			return ctx.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.JSON(rows)
	})
	c.Post("/variables", func(ctx fiber.Ctx) error { return createGeneric(ctx, &SharedVariable{}) })
	c.Patch("/variables/:id", func(ctx fiber.Ctx) error { return patchGeneric(ctx, &SharedVariable{}) })
	c.Delete("/variables/:id", func(ctx fiber.Ctx) error { return deleteGeneric(ctx, &SharedVariable{}) })

	c.Get("/keys", func(ctx fiber.Ctx) error {
		var rows []Key
		if err := db.Order("id desc").Find(&rows).Error; err != nil {
			return ctx.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.JSON(rows)
	})
	c.Post("/keys", func(ctx fiber.Ctx) error { return createGeneric(ctx, &Key{}) })
	c.Delete("/keys/:id", func(ctx fiber.Ctx) error { return deleteGeneric(ctx, &Key{}) })

	c.Get("/api-keys", func(ctx fiber.Ctx) error {
		var rows []ApiKey
		if err := db.Order("id desc").Find(&rows).Error; err != nil {
			return ctx.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.JSON(rows)
	})
	c.Post("/api-keys", func(ctx fiber.Ctx) error { return createGeneric(ctx, &ApiKey{}) })
	c.Delete("/api-keys/:id", func(ctx fiber.Ctx) error { return deleteGeneric(ctx, &ApiKey{}) })

	c.Get("/mcp", func(ctx fiber.Ctx) error {
		var rows []McpEndpoint
		if err := db.Order("id desc").Find(&rows).Error; err != nil {
			return ctx.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.JSON(rows)
	})
	c.Post("/mcp", func(ctx fiber.Ctx) error { return createGeneric(ctx, &McpEndpoint{}) })
	c.Delete("/mcp/:id", func(ctx fiber.Ctx) error { return deleteGeneric(ctx, &McpEndpoint{}) })

	c.Get("/teams", func(ctx fiber.Ctx) error {
		var rows []Team
		if err := db.Preload("Members").Order("id desc").Find(&rows).Error; err != nil {
			return ctx.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return ctx.JSON(rows)
	})
	c.Post("/teams", func(ctx fiber.Ctx) error { return createGeneric(ctx, &Team{}) })
	c.Get("/teams/:id", func(ctx fiber.Ctx) error { return getGeneric(ctx, &Team{}) })
	c.Delete("/teams/:id", func(ctx fiber.Ctx) error { return deleteGeneric(ctx, &Team{}) })
}

// ─── generic helpers ───────────────────────────────────────────────────────

func createGeneric(ctx fiber.Ctx, out interface{}) error {
	if err := ctx.Bind().JSON(out); err != nil {
		return ctx.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	if err := db.Create(out).Error; err != nil {
		return ctx.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.Status(201).JSON(out)
}

func getGeneric(ctx fiber.Ctx, model interface{}) error {
	if err := db.First(model, ctx.Params("id")).Error; err != nil {
		return ctx.Status(404).JSON(fiber.Map{"error": "not found"})
	}
	return ctx.JSON(model)
}

func patchGeneric(ctx fiber.Ctx, model interface{}) error {
	id := ctx.Params("id")
	if err := db.First(model, id).Error; err != nil {
		return ctx.Status(404).JSON(fiber.Map{"error": "not found"})
	}
	if err := ctx.Bind().JSON(model); err != nil {
		return ctx.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	if err := db.Save(model).Error; err != nil {
		return ctx.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(model)
}
func deleteGeneric(ctx fiber.Ctx, model interface{}) error {
	if err := db.Delete(model, ctx.Params("id")).Error; err != nil {
		return ctx.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(fiber.Map{"deleted": ctx.Params("id")})
}