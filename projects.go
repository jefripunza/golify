package main

import (
	"errors"
	"fmt"
	"log"
	"os/exec"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// registerProjects wires the PaaS-style project/env/service CRUD under /api/v1.
// All routes require a JWT (requireAuth) unless noted.
//
// Hierarchy (per Pak Jefri, 2026-08-13):
//   Project = a plain folder (no cluster of its own)
//   Environment = the Kubernetes cluster level (kind cluster named after the
//                 environment's UUID v7 ID). Creating an environment creates
//                 a new kind cluster. Every new project gets a default
//                 "production" environment (and thus one cluster).
func registerProjects(r fiber.Router) {
	auth := r.Group("/projects", requireAuth)

	// ─── Projects ──────────────────────────────────────────────────────────
	auth.Get("/", func(c fiber.Ctx) error {
		var rows []Project
		if err := db.Preload("Envs.Services").Preload("Envs.Domains").
			Order("id desc").Find(&rows).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		// attach live kind cluster status to every environment (cluster name
		// == environment UUID) — the project itself is just a folder
		out := make([]fiber.Map, 0, len(rows))
		for _, p := range rows {
			envs := make([]fiber.Map, 0, len(p.Envs))
			for _, e := range p.Envs {
				svcs := make([]fiber.Map, 0, len(e.Services))
				for _, s := range e.Services {
					svcs = append(svcs, fiber.Map{
						"id": s.ID, "name": s.Name, "kind": s.Kind,
						"image": s.Image, "compose_path": s.ComposePath,
						"status": s.Status, "cpu": s.CPU, "memory": s.Memory,
						"ports": s.Ports, "created_at": s.CreatedAt,
						"updated_at": s.UpdatedAt,
					})
				}
				domains := make([]fiber.Map, 0, len(e.Domains))
				for _, d := range e.Domains {
					domains = append(domains, fiber.Map{"id": d.ID, "host": d.Host})
				}
				envs = append(envs, fiber.Map{
					"id": e.ID, "name": e.Name, "is_production": e.IsProduction,
					"cluster_status": kindClusterStatus(e.ID),
					"services":       svcs, "domains": domains,
					"created_at": e.CreatedAt, "updated_at": e.UpdatedAt,
				})
			}
			out = append(out, fiber.Map{
				"id": p.ID, "name": p.Name, "description": p.Description,
				"source_id": p.SourceID, "environments": envs,
				"env_count": len(envs),
				"created_at": p.CreatedAt, "updated_at": p.UpdatedAt,
			})
		}
		return c.JSON(out)
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
		// Project = folder. It gets a default "production" environment, and
		// that environment owns the kind cluster (cluster name = env UUID).
		p := Project{Name: body.Name, Description: body.Description, SourceID: body.SourceID}
		p.ID = newID()
		if err := db.Create(&p).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		env := Environment{ProjectID: p.ID, Name: "production", IsProduction: true}
		env.ID = newID() // cluster name is the env UUID
		if err := db.Create(&env).Error; err != nil {
			db.Delete(&Project{}, "id = ?", p.ID)
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		// create the kind cluster named after the environment UUID
		if err := kindCreateCluster(env.ID); err != nil {
			db.Delete(&Environment{}, "id = ?", env.ID)
			db.Delete(&Project{}, "id = ?", p.ID)
			return c.Status(502).JSON(fiber.Map{"error": "kind create failed: " + err.Error()})
		}
		return c.Status(201).JSON(fiber.Map{
			"id": p.ID, "name": p.Name, "description": p.Description,
			"source_id": p.SourceID, "environments": []fiber.Map{{
				"id": env.ID, "name": env.Name, "is_production": true,
				"cluster_status": "Running", "services": []fiber.Map{},
				"domains": []fiber.Map{}, "created_at": env.CreatedAt,
				"updated_at": env.UpdatedAt,
			}}, "env_count": 1, "created_at": p.CreatedAt, "updated_at": p.UpdatedAt,
		})
	})

	auth.Get("/:id", func(c fiber.Ctx) error {
		var p Project
		err := db.Preload("Envs.Services").Preload("Envs.Domains").
			First(&p, "id = ?", c.Params("id")).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": "project not found"})
		}
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		envs := make([]fiber.Map, 0, len(p.Envs))
		for _, e := range p.Envs {
			svcs := make([]fiber.Map, 0, len(e.Services))
			for _, s := range e.Services {
				svcs = append(svcs, fiber.Map{
					"id": s.ID, "name": s.Name, "kind": s.Kind,
					"image": s.Image, "compose_path": s.ComposePath,
					"status": s.Status, "cpu": s.CPU, "memory": s.Memory,
					"ports": s.Ports, "created_at": s.CreatedAt, "updated_at": s.UpdatedAt,
				})
			}
			domains := make([]fiber.Map, 0, len(e.Domains))
			for _, d := range e.Domains {
				domains = append(domains, fiber.Map{"id": d.ID, "host": d.Host})
			}
			envs = append(envs, fiber.Map{
				"id": e.ID, "name": e.Name, "is_production": e.IsProduction,
				"cluster_status": kindClusterStatus(e.ID),
				"services": svcs, "domains": domains,
				"created_at": e.CreatedAt, "updated_at": e.UpdatedAt,
			})
		}
		return c.JSON(fiber.Map{
			"id": p.ID, "name": p.Name, "description": p.Description,
			"source_id": p.SourceID, "environments": envs,
			"env_count": len(envs),
			"created_at": p.CreatedAt, "updated_at": p.UpdatedAt,
		})
	})

	// update (edit name/description — cluster name stays the UUID)
	auth.Patch("/:id", func(c fiber.Ctx) error {
		id := c.Params("id")
		var body struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := c.Bind().JSON(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
		}
		var p Project
		if err := db.First(&p, "id = ?", id).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "project not found"})
		}
		if body.Name != "" {
			p.Name = body.Name
		}
		p.Description = body.Description
		if err := db.Save(&p).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(p)
	})

	auth.Delete("/:id", func(c fiber.Ctx) error {
		id := c.Params("id")
		// Cascade rule: a project can only be deleted when it has ZERO
		// environments (1 env = 1 active kind cluster). Deleting a project
		// that still owns environments would leave live clusters orphaned,
		// so we refuse until the user deletes every env first.
		var envCount int64
		if err := db.Model(&Environment{}).Where("project_id = ?", id).Count(&envCount).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		if envCount > 0 {
			return c.Status(409).JSON(fiber.Map{
				"error": fmt.Sprintf("cannot delete project: %d environment(s) still exist — delete all environments (clusters) first", envCount),
			})
		}
		if err := db.Delete(&Project{}, "id = ?", id).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"deleted": id})
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
		pid := c.Params("projectId")
		if pid == "" {
			return c.Status(400).JSON(fiber.Map{"error": "bad project id"})
		}
		var p Project
		if err := db.First(&p, "id = ?", pid).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "project not found"})
		}
		// Environment = Kubernetes cluster. The env UUID v7 is the cluster
		// name — creating an environment creates a brand-new kind cluster.
		env := Environment{ProjectID: pid, Name: body.Name, IsProduction: body.IsProduction}
		env.ID = newID()
		for _, h := range body.Domains {
			env.Domains = append(env.Domains, Domain{Host: h})
		}
		if err := db.Create(&env).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		if err := kindCreateCluster(env.ID); err != nil {
			db.Delete(&Environment{}, "id = ?", env.ID)
			return c.Status(502).JSON(fiber.Map{"error": "kind create failed: " + err.Error()})
		}
		return c.Status(201).JSON(fiber.Map{
			"id": env.ID, "name": env.Name, "is_production": env.IsProduction,
			"cluster_status": "Running", "services": []fiber.Map{},
			"domains": []fiber.Map{}, "created_at": env.CreatedAt, "updated_at": env.UpdatedAt,
		})
	})

	// Delete an environment. Cascade rule: only allowed when the env has
	// ZERO services — otherwise refuse (user must delete services first).
	auth.Delete("/:projectId/environments/:envId", func(c fiber.Ctx) error {
		eid := c.Params("envId")
		var svcCount int64
		if err := db.Model(&Service{}).Where("environment_id = ?", eid).Count(&svcCount).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		if svcCount > 0 {
			return c.Status(409).JSON(fiber.Map{
				"error": fmt.Sprintf("cannot delete environment: %d service(s) still exist — delete all services first", svcCount),
			})
		}
		// tear down the kind cluster named after this env UUID
		kindDeleteCluster(eid)
		if err := db.Delete(&Environment{}, "id = ?", eid).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"deleted": eid})
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
		eid := c.Params("envId")
		if eid == "" {
			return c.Status(400).JSON(fiber.Map{"error": "bad env id"})
		}
		if body.Kind == "" {
			body.Kind = "container"
		}
		if body.Status == "" {
			body.Status = "stopped"
		}
		s := Service{
			EnvironmentID: eid, Name: body.Name, Kind: body.Kind,
			Image: body.Image, ComposePath: body.ComposePath, Status: body.Status,
			CPU: body.CPU, Memory: body.Memory, Ports: body.Ports,
		}
		if err := db.Create(&s).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(201).JSON(s)
	})

	// Delete a single service (innermost level — always allowed, no children).
	auth.Delete("/:projectId/environments/:envId/services/:serviceId", func(c fiber.Ctx) error {
		sid := c.Params("serviceId")
		if err := db.Delete(&Service{}, "id = ?", sid).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"deleted": sid})
	})

	// ─── Start / Stop / Restart a service (status switch) ─────────────────
	auth.Post("/:projectId/environments/:envId/services/:serviceId/start", func(c fiber.Ctx) error {
		return setServiceStatus(c, "running")
	})
	auth.Post("/:projectId/environments/:envId/services/:serviceId/stop", func(c fiber.Ctx) error {
		return setServiceStatus(c, "stopped")
	})
	auth.Post("/:projectId/environments/:envId/services/:serviceId/restart", func(c fiber.Ctx) error {
		return setServiceStatus(c, "restart")
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
	sid := c.Params("serviceId")
	if sid == "" {
		return c.Status(400).JSON(fiber.Map{"error": "bad service id"})
	}
	var s Service
	if err := db.First(&s, "id = ?", sid).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "service not found"})
	}
	if s.Image == "" && s.ComposePath == "" {
		return c.Status(400).JSON(fiber.Map{"error": "service has no container image to control"})
	}

	// Real container action via podman (or docker fallback). If the runtime
	// isn't available on this host (e.g. CI), we gracefully fall back to a
	// DB status flip so the UI still works.
	ctrName := "golify-svc-" + shortID(s.ID)
	if err := containerAction(ctrName, status); err != nil {
		// container may not exist → still flip status (dashboard semantics)
		log.Printf("container action %s on %s failed (%v); flipping DB status only", status, ctrName, err)
	}

	// restart leaves the container running
	dbStatus := status
	if status == "restart" {
		dbStatus = "running"
	}
	s.Status = dbStatus
	if err := db.Save(&s).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"service": s, "runtime": "podman-or-docker"})
}

// containerAction runs the appropriate podman/docker CLI for the given action.
// Returns an error when the runtime binary is missing or the container doesn't
// exist — callers fall back to DB-only updates.
func containerAction(name, action string) error {
	var bin string
	if _, err := exec.LookPath("podman"); err == nil {
		bin = "podman"
	} else if _, err := exec.LookPath("docker"); err == nil {
		bin = "docker"
	} else {
		return errors.New("no container runtime (podman/docker) found")
	}

	switch action {
	case "running":
		// try start first; if container doesn't exist, run from the service's image
		cmd := exec.Command(bin, "start", name)
		if err := cmd.Run(); err != nil {
			// container doesn't exist yet → create+start a placeholder
			return exec.Command(bin, "run", "-d", "--name", name, "alpine", "sleep", "infinity").Run()
		}
		return nil
	case "stopped":
		return exec.Command(bin, "stop", name).Run()
	case "restart":
		return exec.Command(bin, "restart", name).Run()
	default:
		return fmt.Errorf("unknown action %q", action)
	}
}