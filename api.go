package main

import (
	"github.com/gofiber/fiber/v3"
)

func registerAPI(r fiber.Router) {
	// client error reporting — public, forwards to error.sawang.tech
	r.Post("/report/error", reportErrorHandler)

	v1 := r.Group("/v1")

	// public health
	v1.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"app":    "golify",
		})
	})

	// user auth (bcrypt-verified; MUST be declared BEFORE the `auth` group below —
	// Fiber v3 applies an empty-prefix group middleware to every route registered
	// after it, including /auth/login).
	v1.Post("/auth/login", loginHandler)

	// onboarding: public — status + first-admin creation (only while users table is empty)
	v1.Get("/auth/status", authStatusHandler)
	v1.Post("/auth/onboard", onboardHandler)

	// me: JWT-protected — returns the current user; 401 if token invalid.
	// Frontend uses this to detect expired/invalid sessions and auto-redirect
	// to /login instead of silently showing an "Offline" dashboard.
	v1.Get("/auth/me", requireAuth, func(c fiber.Ctx) error {
		claims, ok := c.Locals("claims").(*Claims)
		if !ok || claims == nil {
			return c.Status(401).JSON(fiber.Map{"error": "invalid token"})
		}
		var u User
		if err := db.First(&u, "id = ?", claims.UserID).Error; err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "user not found"})
		}
		return c.JSON(fiber.Map{
			"id":       u.ID,
			"username": u.Username,
			"email":    u.Email,
			"admin":    u.Admin,
		})
	})

	// PaaS-style dashboard CRUD (JWT-protected)
	registerProjects(v1)

	// standalone Domains list (JWT-protected)
	registerDomains(v1)

	// container count (podman/docker) — JWT-protected, cached
	v1.Get("/system/containers", requireAuth, containersHandler)

	// 7 other dashboard menus (servers, sources, s3, variables, keys, api-keys, mcp, teams)
	registerInfra(v1)

	// authenticated routes
	auth := v1.Group("", requireAuth)
	admin := v1.Group("/admin", requireAuth, requireAdmin)

	// auth extras (JWT required)
	auth.Patch("/auth/password", changePassword)
	admin.Post("/auth/register", registerUser)
}
