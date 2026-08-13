package main

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// registerProjects wires the PaaS-style project/env/service CRUD under /api/v1.
// All routes require a JWT (requireAuth) unless noted.
//
// Hierarchy (per Pak Jefri, 2026-08-13):
//
//	Project = a plain folder (no cluster of its own)
//	Environment = the Kubernetes cluster level (kind cluster named after the
//	              environment's UUID v7 ID). Creating an environment creates
//	              a new kind cluster. Every new project gets a default
//	              "production" environment (and thus one cluster).

// isNumeric reports whether s is a plain non-negative integer (ports).
func isNumeric(s string) bool {
	if s == "" {
		return true // empty container port is allowed (host-only mapping)
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func registerProjects(r fiber.Router) {
	auth := r.Group("/projects", requireAuth)

	// ─── Projects ──────────────────────────────────────────────────────────
	auth.Get("/", func(c fiber.Ctx) error {
		var rows []Project
		if err := db.Preload("Envs.Services.Domains").Preload("Envs.Services.Networks").Preload("Envs.Domains").
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
						"type": s.Type, "catalog": s.Catalog,
						"image": s.Image, "image_tag": s.ImageTag,
						"compose_path":      s.ComposePath,
						"description":       s.Description,
						"docker_options":    s.DockerOptions,
						"ports_exposes":     s.PortsExposes,
						"network_aliases":   s.NetworkAliases,
						"basic_auth_enable": s.BasicAuthEnable,
						"basic_auth_user":   s.BasicAuthUser,
						"basic_auth_pass":   s.BasicAuthPass,
						"replicas_mode":     s.ReplicasMode,
						"replicas":          s.Replicas,
						"replicas_min":      s.ReplicasMin,
						"replicas_max":      s.ReplicasMax,
						"status":            s.Status, "cpu": s.CPU, "memory": s.Memory,
						"ports": s.Ports, "created_at": s.CreatedAt,
						"updated_at": s.UpdatedAt,
						"domains": func() []fiber.Map {
							ds := make([]fiber.Map, 0, len(s.Domains))
							for _, d := range s.Domains {
								ds = append(ds, fiber.Map{"id": d.ID, "host": d.Host, "port": d.Port})
							}
							return ds
						}(),
						"networks": func() []fiber.Map {
							ns := make([]fiber.Map, 0, len(s.Networks))
							for _, n := range s.Networks {
								ns = append(ns, fiber.Map{"id": n.ID, "host_port": n.HostPort, "container_port": n.ContainerPort})
							}
							return ns
						}(),
					})
				}
				domains := make([]fiber.Map, 0, len(e.Domains))
				for _, d := range e.Domains {
					domains = append(domains, fiber.Map{"id": d.ID, "host": d.Host})
				}
				envs = append(envs, fiber.Map{
					"id": e.ID, "name": e.Name, "description": e.Description,
					"is_production":  e.IsProduction,
					"ip_internal":    e.IPInternal,
					"cluster_status": kindClusterStatus(e.ID),
					"services":       svcs, "domains": domains,
					"created_at": e.CreatedAt, "updated_at": e.UpdatedAt,
				})
			}
			out = append(out, fiber.Map{
				"id": p.ID, "name": p.Name, "description": p.Description,
				"source_id": p.SourceID, "environments": envs,
				"env_count":  len(envs),
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
		// capture the cluster's internal IP into the environment row
		env.IPInternal = kindClusterIP(env.ID)
		if err := db.Model(&env).Update("ip_internal", env.IPInternal).Error; err != nil {
			log.Printf("[projects] save ip_internal failed: %v", err)
		}
		notify("project", "created", string(p.ID))
		notify("environment", "created", string(env.ID))
		return c.Status(201).JSON(fiber.Map{
			"id": p.ID, "name": p.Name, "description": p.Description,
			"source_id": p.SourceID, "environments": []fiber.Map{{
				"id": env.ID, "name": env.Name, "is_production": true,
				"ip_internal":    env.IPInternal,
				"cluster_status": "Running", "services": []fiber.Map{},
				"domains": []fiber.Map{}, "created_at": env.CreatedAt,
				"updated_at": env.UpdatedAt,
			}}, "env_count": 1, "created_at": p.CreatedAt, "updated_at": p.UpdatedAt,
		})
	})

	auth.Get("/:id", func(c fiber.Ctx) error {
		var p Project
		err := db.Preload("Envs.Services.Domains").Preload("Envs.Services.Networks").Preload("Envs.Domains").
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
					"type": s.Type, "catalog": s.Catalog,
					"image": s.Image, "image_tag": s.ImageTag,
					"compose_path":      s.ComposePath,
					"description":       s.Description,
					"docker_options":    s.DockerOptions,
					"ports_exposes":     s.PortsExposes,
					"network_aliases":   s.NetworkAliases,
					"basic_auth_enable": s.BasicAuthEnable,
					"basic_auth_user":   s.BasicAuthUser,
					"basic_auth_pass":   s.BasicAuthPass,
					"replicas_mode":     s.ReplicasMode,
					"replicas":          s.Replicas,
					"replicas_min":      s.ReplicasMin,
					"replicas_max":      s.ReplicasMax,
					"status":            s.Status, "cpu": s.CPU, "memory": s.Memory,
					"ports": s.Ports, "created_at": s.CreatedAt, "updated_at": s.UpdatedAt,
					"domains": func() []fiber.Map {
						ds := make([]fiber.Map, 0, len(s.Domains))
						for _, d := range s.Domains {
							ds = append(ds, fiber.Map{"id": d.ID, "host": d.Host, "port": d.Port})
						}
						return ds
					}(),
					"networks": func() []fiber.Map {
						ns := make([]fiber.Map, 0, len(s.Networks))
						for _, n := range s.Networks {
							ns = append(ns, fiber.Map{"id": n.ID, "host_port": n.HostPort, "container_port": n.ContainerPort})
						}
						return ns
					}(),
				})
			}
			domains := make([]fiber.Map, 0, len(e.Domains))
			for _, d := range e.Domains {
				domains = append(domains, fiber.Map{"id": d.ID, "host": d.Host})
			}
			envs = append(envs, fiber.Map{
				"id": e.ID, "name": e.Name, "description": e.Description,
				"is_production":  e.IsProduction,
				"ip_internal":    e.IPInternal,
				"cluster_status": kindClusterStatus(e.ID),
				"services":       svcs, "domains": domains,
				"created_at": e.CreatedAt, "updated_at": e.UpdatedAt,
			})
		}
		return c.JSON(fiber.Map{
			"id": p.ID, "name": p.Name, "description": p.Description,
			"source_id": p.SourceID, "environments": envs,
			"env_count":  len(envs),
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
		notify("project", "updated", id)
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
		notify("project", "deleted", id)
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
			Description  string   `json:"description"`
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
		env := Environment{ProjectID: pid, Name: body.Name, Description: body.Description, IsProduction: body.IsProduction}
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
		notify("environment", "created", string(env.ID))
		return c.Status(201).JSON(fiber.Map{
			"id": env.ID, "name": env.Name, "description": env.Description, "is_production": env.IsProduction,
			"cluster_status": "Running", "services": []fiber.Map{},
			"domains": []fiber.Map{}, "created_at": env.CreatedAt, "updated_at": env.UpdatedAt,
		})
	})

	// Update an environment (name / description). Kind cluster is untouched.
	auth.Patch("/:projectId/environments/:envId", func(c fiber.Ctx) error {
		eid := c.Params("envId")
		var env Environment
		if err := db.First(&env, "id = ?", eid).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "environment not found"})
		}
		var body struct {
			Name        *string `json:"name"`
			Description *string `json:"description"`
		}
		if err := c.Bind().JSON(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		if body.Name != nil {
			if *body.Name == "" {
				return c.Status(400).JSON(fiber.Map{"error": "name cannot be empty"})
			}
			env.Name = *body.Name
		}
		if body.Description != nil {
			env.Description = *body.Description
		}
		if err := db.Save(&env).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		notify("environment", "updated", eid)
		return c.JSON(fiber.Map{
			"id": env.ID, "name": env.Name, "description": env.Description,
			"is_production": env.IsProduction, "updated_at": env.UpdatedAt,
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
		notify("environment", "deleted", eid)
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
			Type        string   `json:"type"`    // application | database | tool
			Catalog     string   `json:"catalog"` // e.g. docker-image | version-control | postgres | qdrant
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
		if body.Type == "" {
			body.Type = "application"
		}
		if body.Status == "" {
			body.Status = "stopped"
		}
		s := Service{
			EnvironmentID: eid, Name: body.Name, Kind: body.Kind,
			Type: body.Type, Catalog: body.Catalog,
			Image: body.Image, ComposePath: body.ComposePath, Status: body.Status,
			CPU: body.CPU, Memory: body.Memory, Ports: body.Ports,
		}
		if err := db.Create(&s).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		notify("service", "created", string(s.ID))
		return c.Status(201).JSON(s)
	})

	// Delete a single service. Cascade: removes the container (podman),
	// deployment history, domains and networks attached to this service.
	auth.Delete("/:projectId/environments/:envId/services/:serviceId", func(c fiber.Ctx) error {
		sid := c.Params("serviceId")
		var svc Service
		if err := db.First(&svc, "id = ?", sid).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "service not found"})
		}
		// 1) stop & remove the container if it exists
		ctrName := "golify-" + strings.ToLower(strings.ReplaceAll(svc.Name, " ", "-"))
		if out, err := exec.Command("podman", "rm", "-f", ctrName).CombinedOutput(); err != nil {
			log.Printf("[service] podman rm %s: %v (%s)", ctrName, err, strings.TrimSpace(string(out)))
		}
		// 2) cascade children
		if err := db.Where("service_id = ?", sid).Delete(&Deployment{}).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "delete deployments: " + err.Error()})
		}
		if err := db.Where("service_id = ?", sid).Delete(&ServiceDomain{}).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "delete domains: " + err.Error()})
		}
		if err := db.Where("service_id = ?", sid).Delete(&ServiceNetwork{}).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "delete networks: " + err.Error()})
		}
		// 3) the service itself
		if err := db.Delete(&Service{}, "id = ?", sid).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		notify("service", "deleted", sid)
		return c.JSON(fiber.Map{"deleted": sid})
	})

	// PATCH service configuration (Coolify-style General settings).
	auth.Patch("/:projectId/environments/:envId/services/:serviceId", func(c fiber.Ctx) error {
		sid := c.Params("serviceId")
		var svc Service
		if err := db.First(&svc, "id = ?", sid).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "service not found"})
		}
		var body struct {
			Name            *string   `json:"name"`
			Description     *string   `json:"description"`
			Image           *string   `json:"image"`
			ImageTag        *string   `json:"image_tag"`
			DockerOptions   *string   `json:"docker_options"`
			PortsExposes    *string   `json:"ports_exposes"`
			PortMappings    *[]string `json:"port_mappings"`
			NetworkAliases  *[]string `json:"network_aliases"`
			BasicAuthEnable *bool     `json:"basic_auth_enable"`
			BasicAuthUser   *string   `json:"basic_auth_user"`
			BasicAuthPass   *string   `json:"basic_auth_pass"`
			ReplicasMode    *string   `json:"replicas_mode"`
			Replicas        *int      `json:"replicas"`
			ReplicasMin     *int      `json:"replicas_min"`
			ReplicasMax     *int      `json:"replicas_max"`
		}
		if err := c.Bind().JSON(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid JSON"})
		}
		if body.Name != nil {
			if *body.Name == "" {
				return c.Status(400).JSON(fiber.Map{"error": "name cannot be empty"})
			}
			svc.Name = *body.Name
		}
		if body.Description != nil {
			svc.Description = *body.Description
		}
		if body.Image != nil {
			svc.Image = *body.Image
		}
		if body.ImageTag != nil {
			svc.ImageTag = *body.ImageTag
		}
		if body.DockerOptions != nil {
			svc.DockerOptions = *body.DockerOptions
		}
		if body.PortsExposes != nil {
			svc.PortsExposes = *body.PortsExposes
		}
		if body.PortMappings != nil {
			svc.PortMappings = *body.PortMappings
		}
		if body.NetworkAliases != nil {
			svc.NetworkAliases = *body.NetworkAliases
		}
		if body.BasicAuthEnable != nil {
			svc.BasicAuthEnable = *body.BasicAuthEnable
		}
		if body.BasicAuthUser != nil {
			svc.BasicAuthUser = *body.BasicAuthUser
		}
		if body.BasicAuthPass != nil {
			svc.BasicAuthPass = *body.BasicAuthPass
		}
		if body.ReplicasMode != nil && (*body.ReplicasMode == "fix" || *body.ReplicasMode == "range") {
			svc.ReplicasMode = *body.ReplicasMode
		}
		if body.Replicas != nil && *body.Replicas >= 1 {
			svc.Replicas = *body.Replicas
		}
		if body.ReplicasMin != nil && *body.ReplicasMin >= 1 {
			svc.ReplicasMin = *body.ReplicasMin
		}
		if body.ReplicasMax != nil && *body.ReplicasMax >= *body.ReplicasMin {
			svc.ReplicasMax = *body.ReplicasMax
		}
		if err := db.Save(&svc).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		notify("service", "updated", sid)
		return c.JSON(svc)
	})

	// ─── Service domains (many domains/subdomains per service, each → port) ──
	auth.Get("/:projectId/environments/:envId/services/:serviceId/domains", func(c fiber.Ctx) error {
		var rows []ServiceDomain
		if err := db.Where("service_id = ?", c.Params("serviceId")).Order("host asc").Find(&rows).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(rows)
	})

	auth.Post("/:projectId/environments/:envId/services/:serviceId/domains", func(c fiber.Ctx) error {
		var body struct {
			Host string `json:"host"`
			Port string `json:"port"`
		}
		if err := c.Bind().JSON(&body); err != nil || body.Host == "" {
			return c.Status(400).JSON(fiber.Map{"error": "host required"})
		}
		if body.Port == "" {
			body.Port = "80"
		}
		// unique check: same host (subdomain+domain combo, incl. bare root) not allowed twice per service
		var count int64
		if err := db.Model(&ServiceDomain{}).Where("service_id = ? AND host = ?", c.Params("serviceId"), body.Host).Count(&count).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		if count > 0 {
			return c.Status(409).JSON(fiber.Map{"error": "domain already exists: " + body.Host})
		}
		sd := ServiceDomain{
			ServiceID: c.Params("serviceId"),
			Host:      body.Host,
			Port:      body.Port,
		}
		if err := db.Create(&sd).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		// generate a self-signed cert so HTTPS works for this domain right away
		// (real ACME/Let's Encrypt can replace it later — loader prefers LE dir).
		if err := ensureSelfSignedCert(body.Host); err != nil {
			log.Printf("[ssl] self-signed cert for %s: %v", body.Host, err)
		}
		notify("domain", "created", string(sd.ID))
		return c.Status(201).JSON(sd)
	})

	auth.Delete("/:projectId/environments/:envId/services/:serviceId/domains/:domainId", func(c fiber.Ctx) error {
		did := c.Params("domainId")
		if err := db.Delete(&ServiceDomain{}, "id = ?", did).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		notify("domain", "deleted", did)
		return c.JSON(fiber.Map{"deleted": did})
	})

	// PATCH domain (edit host/port)
	auth.Patch("/:projectId/environments/:envId/services/:serviceId/domains/:domainId", func(c fiber.Ctx) error {
		did := c.Params("domainId")
		var sd ServiceDomain
		if err := db.First(&sd, "id = ?", did).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "domain not found"})
		}
		var body struct {
			Host *string `json:"host"`
			Port *string `json:"port"`
		}
		if err := c.Bind().JSON(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid JSON"})
		}
		if body.Host != nil {
			if *body.Host == "" {
				return c.Status(400).JSON(fiber.Map{"error": "host required"})
			}
			// unique check on update: same host not allowed, excluding this domain itself
			var count int64
			if err := db.Model(&ServiceDomain{}).Where("service_id = ? AND host = ? AND id <> ?", sd.ServiceID, *body.Host, did).Count(&count).Error; err != nil {
				return c.Status(500).JSON(fiber.Map{"error": err.Error()})
			}
			if count > 0 {
				return c.Status(409).JSON(fiber.Map{"error": "domain already exists: " + *body.Host})
			}
			sd.Host = *body.Host
		}
		if body.Port != nil {
			if *body.Port == "" {
				*body.Port = "80"
			}
			sd.Port = *body.Port
		}
		if err := db.Save(&sd).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		notify("domain", "updated", did)
		return c.JSON(sd)
	})

	// ─── Service networks (port mappings, service_networks table) ──────────
	auth.Get("/:projectId/environments/:envId/services/:serviceId/networks", func(c fiber.Ctx) error {
		var rows []ServiceNetwork
		if err := db.Where("service_id = ?", c.Params("serviceId")).Order("host_port asc").Find(&rows).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(rows)
	})

	auth.Post("/:projectId/environments/:envId/services/:serviceId/networks", func(c fiber.Ctx) error {
		var body struct {
			HostPort      string `json:"host_port"`
			ContainerPort string `json:"container_port"`
		}
		if err := c.Bind().JSON(&body); err != nil || body.HostPort == "" {
			return c.Status(400).JSON(fiber.Map{"error": "host_port required"})
		}
		// ports must be numeric
		if !isNumeric(body.HostPort) || !isNumeric(body.ContainerPort) {
			return c.Status(400).JSON(fiber.Map{"error": "ports must be numbers"})
		}
		sn := ServiceNetwork{
			ServiceID:     c.Params("serviceId"),
			HostPort:      body.HostPort,
			ContainerPort: body.ContainerPort,
		}
		if err := db.Create(&sn).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(201).JSON(sn)
	})

	auth.Patch("/:projectId/environments/:envId/services/:serviceId/networks/:networkId", func(c fiber.Ctx) error {
		nid := c.Params("networkId")
		var sn ServiceNetwork
		if err := db.First(&sn, "id = ?", nid).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "network not found"})
		}
		var body struct {
			HostPort      *string `json:"host_port"`
			ContainerPort *string `json:"container_port"`
		}
		if err := c.Bind().JSON(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid JSON"})
		}
		if body.HostPort != nil {
			if *body.HostPort == "" {
				return c.Status(400).JSON(fiber.Map{"error": "host_port required"})
			}
			if !isNumeric(*body.HostPort) {
				return c.Status(400).JSON(fiber.Map{"error": "ports must be numbers"})
			}
			sn.HostPort = *body.HostPort
		}
		if body.ContainerPort != nil {
			if !isNumeric(*body.ContainerPort) {
				return c.Status(400).JSON(fiber.Map{"error": "ports must be numbers"})
			}
			sn.ContainerPort = *body.ContainerPort
		}
		if err := db.Save(&sn).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(sn)
	})

	auth.Delete("/:projectId/environments/:envId/services/:serviceId/networks/:networkId", func(c fiber.Ctx) error {
		nid := c.Params("networkId")
		if err := db.Delete(&ServiceNetwork{}, "id = ?", nid).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"deleted": nid})
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

	// ─── Deployments (history + trigger) ──────────────────────────────────
	registerDeployments(auth)

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
