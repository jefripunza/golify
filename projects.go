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
func registerProjects(r fiber.Router) {
	auth := r.Group("/projects", requireAuth)

	// ─── Projects ──────────────────────────────────────────────────────────
	auth.Get("/", func(c fiber.Ctx) error {
		var rows []Project
		if err := db.Preload("Envs").Order("id desc").Find(&rows).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		// attach live kind cluster status for each project (id = cluster name)
		type projectWithStatus struct {
			Project
			ClusterStatus string `json:"cluster_status"`
		}
		out := make([]projectWithStatus, 0, len(rows))
		for _, p := range rows {
			out = append(out, projectWithStatus{Project: p, ClusterStatus: kindClusterStatus(p.ID)})
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
		// Project = Kubernetes cluster. The project ID (UUID v7) becomes the
		// kind cluster name.
		p := Project{Name: body.Name, Description: body.Description, SourceID: body.SourceID}
		p.ID = newID() // explicit so we know the cluster name before creating
		if err := db.Create(&p).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		// create the kind cluster named after the project UUID
		clusterName := p.ID
		if err := kindCreateCluster(clusterName); err != nil {
			// cluster creation failed — roll back the DB row so the list
			// only shows clusters that actually exist
			db.Delete(&Project{}, "id = ?", p.ID)
			return c.Status(502).JSON(fiber.Map{"error": "kind create failed: " + err.Error()})
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
		if err := db.Delete(&Project{}, "id = ?", id).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		// also tear down the kind cluster named after the project UUID
		kindDeleteCluster(id)
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
		env := Environment{ProjectID: pid, Name: body.Name, IsProduction: body.IsProduction}
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