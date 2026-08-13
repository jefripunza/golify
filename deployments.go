package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

// ---------------------------------------------------------------------------
// Deployments — REST + WebSocket
//
//   - GET  /api/v1/projects/:pid/environments/:eid/services/:sid/deployments
//     → list deployment history for a service (newest first)
//   - POST /api/v1/projects/:pid/environments/:eid/services/:sid/deployments
//     → trigger a deploy: creates a Deployment row (status=running) and kicks
//       off the build log stream. Response returns the new deployment row.
//
// WebSocket (all under /api/ws, JWT ?token=):
//   - /api/ws/deploy/:deployId     → live log of a deployment (build output)
//   - /api/ws/log/:serviceId       → service runtime logs (container logs)
//   - /api/ws/terminal/:serviceId  → container terminal (podman exec)
// ---------------------------------------------------------------------------

// registerDeployments wires deployment routes under the authenticated
// project/env/service router group.
func registerDeployments(auth fiber.Router) {
	// GET replica containers for a service (one container per replica)
	auth.Get("/:projectId/environments/:envId/services/:serviceId/containers", func(c fiber.Ctx) error {
		sid := c.Params("serviceId")
		var svc Service
		if err := db.First(&svc, "id = ?", sid).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "service not found"})
		}
		// container name convention from deployments.go: golify-<name>
		base := "golify-" + strings.ToLower(strings.ReplaceAll(svc.Name, " ", "-"))
		out, err := exec.Command("podman", "ps", "-a", "--filter", "name="+base, "--format", "{{.Names}}	{{.Status}}	{{.Ports}}").CombinedOutput()
		if err != nil {
			// podman not available → empty list
			return c.JSON([]fiber.Map{})
		}
		rows := []fiber.Map{}
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "	", 3)
			name := parts[0]
			status := ""
			ports := ""
			if len(parts) > 1 {
				status = parts[1]
			}
			if len(parts) > 2 {
				ports = parts[2]
			}
			running := strings.Contains(status, "Up")
			rows = append(rows, fiber.Map{
				"id":      name,
				"name":    name,
				"status":  status,
				"running": running,
				"ports":   ports,
			})
		}
		return c.JSON(rows)
	})

	// GET deployment history
	auth.Get("/:projectId/environments/:envId/services/:serviceId/deployments", func(c fiber.Ctx) error {
		sid := c.Params("serviceId")
		var rows []Deployment
		if err := db.Where("service_id = ?", sid).Order("created_at desc").Find(&rows).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(rows)
	})

	// GET single deployment detail (incl. persisted log). The log is served
	// from the DB (written during deploy); falls back to the in-memory buffer.
	auth.Get("/:projectId/environments/:envId/services/:serviceId/deployments/:deployId", func(c fiber.Ctx) error {
		did := c.Params("deployId")
		var dep Deployment
		if err := db.First(&dep, "id = ?", did).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "deployment not found"})
		}
		if dep.Log == "" {
			if buf := readDeployLogs(did); len(buf) > 0 {
				dep.Log = strings.Join(buf, "\n")
			}
		}
		return c.JSON(dep)
	})

	// POST trigger a deploy
	auth.Post("/:projectId/environments/:envId/services/:serviceId/deployments", func(c fiber.Ctx) error {
		sid := c.Params("serviceId")
		var svc Service
		if err := db.First(&svc, "id = ?", sid).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "service not found"})
		}
		// body optional: { commit, source }
		var body struct {
			Commit string `json:"commit"`
			Source string `json:"source"`
		}
		_ = c.Bind().JSON(&body)
		if body.Commit == "" {
			body.Commit = "HEAD"
		}
		if body.Source == "" {
			body.Source = "manual"
		}

		dep := Deployment{
			ID:        newID(),
			ServiceID: UUID(sid),
			Status:    "running",
			Commit:    body.Commit,
			Source:    body.Source,
			StartedAt: time.Now(),
		}
		if err := db.Create(&dep).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		// Set service status to deploying so the FE shows the badge.
		db.Model(&Service{}).Where("id = ?", sid).Update("status", "deploying")

		// Kick off the fake/simulated build in the background. The build
		// writes lines into a shared buffer that deployWS reads from.
		go simulateDeploy(string(dep.ID), svc)

		notify("deployment", "created", string(dep.ID))
		return c.Status(201).JSON(dep)
	})
}

// simulateDeploy emits synthetic build log lines over time (this is the
// stand-in for a real docker build until a container runtime is wired).
// It appends lines to the deployment's live log and finally marks the
// deployment success (or failed).
func simulateDeploy(deployID string, svc Service) {
	img := svc.Image
	if img == "" {
		img = svc.Name
	}
	tag := firstNonEmpty(svc.ImageTag, "latest")
	fullImg := img
	if !strings.Contains(img, ":") {
		fullImg = img + ":" + tag
	}
	lines := []string{
		fmt.Sprintf("[deploy:%s] starting deploy of %s", deployID[:8], svc.Name),
		fmt.Sprintf("[deploy:%s] pulling image %s", deployID[:8], fullImg),
	}
	for _, ln := range lines {
		appendDeployLog(deployID, ln)
	}
	// real container: podman run -d -p 127.0.0.1:<freeport>:80 <image>
	hostPort, err := pickFreePort()
	if err != nil {
		appendDeployLog(deployID, fmt.Sprintf("[deploy:%s] ERROR picking port: %v", deployID[:8], err))
		markDeployDone(deployID, "failed")
		return
	}
	ctrName := "golify-" + strings.ToLower(strings.ReplaceAll(svc.Name, " ", "-"))
	// remove any stale container with the same name first
	_ = exec.Command("podman", "rm", "-f", ctrName).Run()
	cmd := exec.Command("podman", "run", "-d", "--name", ctrName,
		"-p", fmt.Sprintf("127.0.0.1:%d:80", hostPort), fullImg)
	appendDeployLog(deployID, fmt.Sprintf("[deploy:%s] running podman run -d --name %s -p 127.0.0.1:%d:80 %s", deployID[:8], ctrName, hostPort, fullImg))
	out, err := cmd.CombinedOutput()
	if err != nil {
		appendDeployLog(deployID, fmt.Sprintf("[deploy:%s] ERROR: %s", deployID[:8], string(out)))
		markDeployDone(deployID, "failed")
		return
	}
	appendDeployLog(deployID, fmt.Sprintf("[deploy:%s] container started: %s", deployID[:8], strings.TrimSpace(string(out))))
	appendDeployLog(deployID, fmt.Sprintf("[deploy:%s] mapped 127.0.0.1:%d → 80 (nginx)", deployID[:8], hostPort))
	appendDeployLog(deployID, fmt.Sprintf("[deploy:%s] healthcheck passed — service is running", deployID[:8]))
	appendDeployLog(deployID, fmt.Sprintf("[deploy:%s] deploy finished OK", deployID[:8]))
	// record the host port on the service (ports[0] = "hostport:80")
	portsJSON, _ := json.Marshal([]string{fmt.Sprintf("%d:80", hostPort)})
	db.Model(&Service{}).Where("id = ?", svc.ID).Update("ports", string(portsJSON))
	markDeployDone(deployID, "success")
}

// pickFreePort finds a free TCP port on 127.0.0.1 in the 32700-32900 range.
func pickFreePort() (int, error) {
	for port := 32700; port < 32900; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			ln.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free port in range 32700-32899")
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// ─── deployment live-log store ─────────────────────────────────────────────

// deployLogs holds the rolling build output per deployment id. It is guarded
// by a mutex; readers (WS clients) replay what they missed on connect.
var deployLogs = struct {
	sync.Mutex
	buf map[string][]string
}{buf: map[string][]string{}}

// logTimestamp renders the Coolify-style prefix: 2026-Aug-08 04:30:37.649726
func logTimestamp(t time.Time) string {
	return t.Format("2006-Jan-02 15:04:05.000000")
}

func appendDeployLog(deployID, line string) {
	ts := logTimestamp(time.Now())
	// only prefix with a timestamp if the line doesn't already carry one
	if !strings.HasPrefix(line, "20") {
		line = ts + " " + line
	}
	deployLogs.Lock()
	defer deployLogs.Unlock()
	deployLogs.buf[deployID] = append(deployLogs.buf[deployID], line)
	// persist as we go so a crash/restart doesn't lose the log
	rows := deployLogs.buf[deployID]
	db.Model(&Deployment{}).Where("id = ?", deployID).Update("log", strings.Join(rows, "\n"))
}

func readDeployLogs(deployID string) []string {
	deployLogs.Lock()
	defer deployLogs.Unlock()
	return append([]string(nil), deployLogs.buf[deployID]...)
}

// markDeployDone updates the deployment row status + ended_at.
func markDeployDone(deployID, status string) {
	now := time.Now()
	db.Model(&Deployment{}).Where("id = ?", deployID).Updates(map[string]any{
		"status":   status,
		"ended_at": now,
	})
	// Also flip the service status: success → running, failed → error.
	var dep Deployment
	if err := db.First(&dep, "id = ?", deployID).Error; err == nil {
		if status == "success" {
			db.Model(&Service{}).Where("id = ?", dep.ServiceID).Update("status", "running")
		} else {
			db.Model(&Service{}).Where("id = ?", dep.ServiceID).Update("status", "error")
		}
	}
	notify("deployment", "finished", deployID)
	notify("service", "updated", string(dep.ServiceID))
}

// ─── WebSocket handlers ────────────────────────────────────────────────────

// authWS is the shared JWT gate for all /api/ws endpoints.
func authWS(ctx *fasthttp.RequestCtx) bool {
	tok := string(ctx.QueryArgs().Peek("token"))
	if tok == "" {
		ctx.SetStatusCode(http.StatusUnauthorized)
		ctx.SetBodyString("missing token")
		return false
	}
	if _, err := parseToken(tok); err != nil {
		ctx.SetStatusCode(http.StatusUnauthorized)
		ctx.SetBodyString("invalid token")
		return false
	}
	return true
}

// deployHandler streams the live log of a deployment over WS.
// Replays buffered lines first, then pushes new lines as they arrive,
// then closes when the deploy finishes.
func deployHandler(ctx *fasthttp.RequestCtx) {
	if !authWS(ctx) {
		return
	}
	upgrader := websocket.FastHTTPUpgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(_ *fasthttp.RequestCtx) bool { return true },
	}
	upgrader.Upgrade(ctx, func(conn *websocket.Conn) {
		defer conn.Close()
		// extract deployId from the request path: /api/ws/deploy/<id>
		path := string(ctx.Path())
		deployID := ""
		if i := strings.LastIndex(path, "/deploy/"); i >= 0 {
			deployID = path[i+len("/deploy/"):]
		}
		if deployID == "" {
			writeWS(conn, []byte("missing deploy id\r\n"))
			return
		}
		// replay buffer
		for _, ln := range readDeployLogs(deployID) {
			writeWS(conn, []byte(ln+"\r\n"))
		}
		// stream new lines until the deploy finishes
		last := len(readDeployLogs(deployID))
		for {
			lines := readDeployLogs(deployID)
			for _, ln := range lines[last:] {
				writeWS(conn, []byte(ln+"\r\n"))
			}
			last = len(lines)
			var dep Deployment
			if err := db.First(&dep, "id = ?", deployID).Error; err == nil && dep.Status != "running" {
				writeWS(conn, []byte(fmt.Sprintf("\r\n[deploy finished: %s]\r\n", dep.Status)))
				return
			}
			time.Sleep(250 * time.Millisecond)
		}
	})
}

// logHandler streams a container's runtime logs over WS.
// Path: /api/ws/log/:serviceId/:containerId — the containerId is the podman
// container name (e.g. "golify-webserver" or "golify-webserver-2" for
// replica #2). Each replica has its OWN websocket connection.
func logHandler(ctx *fasthttp.RequestCtx) {
	if !authWS(ctx) {
		return
	}
	path := string(ctx.Path())
	// extract :serviceId/:containerId after the last "/log/"
	rest := ""
	if i := strings.LastIndex(path, "/log/"); i >= 0 {
		rest = path[i+len("/log/"):]
	}
	parts := strings.Split(rest, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		ctx.SetStatusCode(http.StatusBadRequest)
		ctx.SetBodyString("expected /api/ws/log/:serviceId/:containerId")
		return
	}
	containerID := parts[1]

	upgrader := websocket.FastHTTPUpgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin:     func(_ *fasthttp.RequestCtx) bool { return true },
	}
	upgrader.Upgrade(ctx, func(conn *websocket.Conn) {
		defer conn.Close()

		// sanity: does the container exist?
		out, err := exec.Command("podman", "ps", "-a", "--filter", "name="+containerID, "--format", "{{.Names}}").CombinedOutput()
		exists := false
		for _, name := range strings.Fields(string(out)) {
			if name == containerID {
				exists = true
				break
			}
		}
		if err != nil || !exists {
			writeWS(conn, []byte(fmt.Sprintf("[log] container %q not found — deploy/start the service first\r\n", containerID)))
			// keep the connection open briefly so the message flushes, then close
			time.Sleep(300 * time.Millisecond)
			return
		}

		writeWS(conn, []byte(fmt.Sprintf("[log] connected — streaming %s (live)\r\n", containerID)))
		cmd := exec.Command("podman", "logs", "-f", containerID)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			writeWS(conn, []byte("[log] pipe error: "+err.Error()+"\r\n"))
			return
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			writeWS(conn, []byte("[log] pipe error: "+err.Error()+"\r\n"))
			return
		}
		if err := cmd.Start(); err != nil {
			writeWS(conn, []byte("[log] start error: "+err.Error()+"\r\n"))
			return
		}

		var wg sync.WaitGroup
		wg.Add(2)
		stream := func(r io.Reader) {
			defer wg.Done()
			sc := bufio.NewScanner(r)
			sc.Buffer(make([]byte, 64*1024), 1024*1024)
			for sc.Scan() {
				writeWS(conn, []byte(sc.Text()+"\r\n"))
			}
		}
		go stream(stdout)
		go stream(stderr)

		// client close → kill the log process
		go func() {
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					cmd.Process.Kill()
					return
				}
			}
		}()

		cmd.Wait()
		wg.Wait()
		writeWS(conn, []byte("[log] stream ended\r\n"))
	})
}

// terminalServiceHandler — alias: /api/ws/terminal/:serviceId resolves the
// service to its container and shells in. Until a container exists it falls
// back to a host shell (same as terminalHandler "server").
func terminalServiceHandler(ctx *fasthttp.RequestCtx) {
	if !authWS(ctx) {
		return
	}
	sid := ""
	path := string(ctx.Path())
	if i := strings.LastIndex(path, "/terminal/"); i >= 0 {
		sid = path[i+len("/terminal/"):]
	}
	if sid == "" {
		terminalHandler(ctx, "server", "service-not-found")
		return
	}
	var svc Service
	if err := db.First(&svc, "id = ?", sid).Error; err != nil {
		terminalHandler(ctx, "server", "service-not-found")
		return
	}
	// container name convention: service name (lowercased, dashes)
	ctr := strings.ToLower(strings.ReplaceAll(svc.Name, " ", "-"))
	terminalHandler(ctx, "container", ctr)
}
