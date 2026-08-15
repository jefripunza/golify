package main

import (
	"io"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

// WebSocket terminal exec — served on the SAME port as the API/SPA (unified).
//   1. authenticates via JWT in the `?token=` query string
//   2. resolves the requested service to a podman container (or host shell)
//   3. spawns `podman exec -it <ctr> /bin/sh` (fallback host /bin/sh) with
//      stdin/stdout piped to the WS connection
//
// Fiber v3 has no first-class WebSocket support, so the unified fasthttp
// server routes `/ws/*` to this handler and everything else to Fiber.

func streamHandler(ctx *fasthttp.RequestCtx) {
	tok := string(ctx.QueryArgs().Peek("token"))
	if tok == "" {
		ctx.SetStatusCode(http.StatusUnauthorized)
		ctx.SetBodyString("missing token")
		return
	}
	if _, err := parseToken(tok); err != nil {
		ctx.SetStatusCode(http.StatusUnauthorized)
		ctx.SetBodyString("invalid token")
		return
	}

	upgrader := websocket.FastHTTPUpgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(_ *fasthttp.RequestCtx) bool { return true },
	}
	// streamHandlerUpgraded runs in the WS goroutine after upgrade.
	upgrader.Upgrade(ctx, func(conn *websocket.Conn) {
		sid := string(ctx.QueryArgs().Peek("service_id"))
		if sid == "" {
			sid = "host"
		}
		cmd := buildExecCmd(sid)

		stdin, err := cmd.StdinPipe()
		if err != nil {
			return
		}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return
		}
		if err := cmd.Start(); err != nil {
			writeWS(conn, []byte("\r\n\x1b[31mfailed: "+err.Error()+"\x1b[0m\r\n"))
			return
		}

		// WS frames → subprocess stdin
		go func() {
			for {
				_, msg, rerr := conn.ReadMessage()
				if rerr != nil {
					break
				}
				stdin.Write(msg)
			}
		}()

		// subprocess stdout+stderr → WS
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			mw := &wsWriter{conn: conn}
			io.Copy(mw, io.MultiReader(stdout, stderr))
		}()

		cmd.Wait()
		// Use the SAME lock as wsWriter so this write never races with
		// io.Copy above (previously a bare WriteMessage → panic:
		// "concurrent write to websocket connection" killed the process).
		writeWS(conn, []byte("\r\n\x1b[33m[process exited]\x1b[0m\r\n"))
		wg.Wait()
	})
}

// wsRequestHandler handles /api/ws/* paths on the backend port.
// (Legacy /ws/* also accepted for backward compatibility.)
// Returns true if the path was handled (WS or /api/ws/health), false otherwise.
func wsRequestHandler(ctx *fasthttp.RequestCtx) bool {
	path := string(ctx.Path())
	// new canonical prefix: /api/ws/... (per user rule — WS lives under /api)
	apiPrefix := "/api/ws"
	legacyPrefix := "/ws"
	if !strings.HasPrefix(path, apiPrefix) && !strings.HasPrefix(path, legacyPrefix) {
		return false
	}
	// normalize to the legacy path for the handler switch below
	rel := strings.TrimPrefix(path, apiPrefix)
	rel = strings.TrimPrefix(rel, legacyPrefix)
	switch {
	case rel == "/health" || rel == "":
		ctx.SetStatusCode(200)
		ctx.SetBodyString(`{"app":"golify-ws","status":"ok"}`)
		return true
	case rel == "/analytic":
		analyticHandler(ctx)
		return true
	case rel == "/realtime":
		realtimeHandler(ctx)
		return true
	}
	// /api/ws/terminal/server/:serverId — host terminal for a registered server
	if serverID, ok := matchWSPath(rel, "/terminal/server/"); ok {
		terminalHandler(ctx, "server", serverID)
		return true
	}
	// /api/ws/terminal/container/:containerId — container terminal (podman exec)
	if containerID, ok := matchWSPath(rel, "/terminal/container/"); ok {
		terminalHandler(ctx, "container", containerID)
		return true
	}
	// /api/ws/deploy/:deployId — live build log of a deployment
	if deployID, ok := matchWSPath(rel, "/deploy/"); ok {
		_ = deployID // deployHandler reads it from the path itself
		deployHandler(ctx)
		return true
	}
	// /api/ws/log/:serviceId/:containerId — service runtime logs (per replica)
	// matchWSPath would reject ids containing "/", so route by prefix and let
	// logHandler parse the container id from the path itself.
	if strings.HasPrefix(rel, "/log/") {
		logHandler(ctx)
		return true
	}
	// /api/ws/terminal/:serviceId/:containerId — container terminal (podman
	// exec) for a SPECIFIC replica/container, OR /api/ws/terminal/:serviceId
	// for the legacy single-terminal path. terminalServiceHandler parses the
	// segments itself (container id = last path segment when present).
	if strings.HasPrefix(rel, "/terminal/") {
		terminalServiceHandler(ctx)
		return true
	}
	if strings.HasPrefix(rel, "/") {
		streamHandler(ctx)
		return true
	}
	return false
}

// matchWSPath extracts the id segment after prefix, or ("", false).
func matchWSPath(path, prefix string) (string, bool) {
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	id := strings.TrimPrefix(path, prefix)
	if id == "" || strings.Contains(id, "/") {
		return "", false
	}
	return id, true
}

// serveUnified serves a fiber.App and the WebSocket handler on the SAME
// listener/port: /ws/* goes to the WS handler, everything else goes to Fiber.
func serveUnified(ln net.Listener, fiberHandler fasthttp.RequestHandler) error {
	srv := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			if wsRequestHandler(ctx) {
				return
			}
			fiberHandler(ctx)
		},
		Logger: discardLogger{},
	}
	return srv.Serve(ln)
}

func buildExecCmd(sid string) *exec.Cmd {
	if sid == "host" {
		return exec.Command("/bin/sh")
	}
	if _, err := exec.LookPath("podman"); err == nil {
		return exec.Command("podman", "exec", "-it", "golify-svc-"+sid, "/bin/sh")
	}
	if _, err := exec.LookPath("docker"); err == nil {
		return exec.Command("docker", "exec", "-it", "golify-svc-"+sid, "/bin/sh")
	}
	return exec.Command("/bin/sh")
}

type wsWriter struct {
	conn *websocket.Conn
}

// Per-connection write mutexes. fasthttp/websocket panics on concurrent
// WriteMessage calls to the SAME conn, so we need one lock per conn — but a
// GLOBAL lock is deadly: one stuck connection (e.g. a Vite dev-proxy WS that
// half-closed into FIN-WAIT-2) blocks ALL websocket traffic on the server
// (terminal, logs, realtime). With per-conn locks a stuck conn only freezes
// itself.
var wsMuByConn sync.Map // *websocket.Conn -> *sync.Mutex

func wsLockFor(conn *websocket.Conn) *sync.Mutex {
	mu, _ := wsMuByConn.LoadOrStore(conn, &sync.Mutex{})
	return mu.(*sync.Mutex)
}

// writeWS serializes writes to a SINGLE ws conn and enforces a write
// deadline so a dead/half-closed conn cannot hang the writer forever.
func writeWS(conn *websocket.Conn, p []byte) error {
	mu := wsLockFor(conn)
	mu.Lock()
	defer mu.Unlock()
	// WriteMessage panics if the conn is closed underneath us — catch it.
	defer func() {
		if r := recover(); r != nil {
			// ignore: client disconnected
		}
	}()
	_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return conn.WriteMessage(websocket.TextMessage, p)
}

func (w *wsWriter) Write(p []byte) (int, error) {
	if err := writeWS(w.conn, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

// startWSServer removed — WS is now served on the same unified port as the
// API/SPA via wsRequestHandler + unifiedHandler in main.go.

// discardLogger silences fasthttp access logs.
type discardLogger struct{}

func (discardLogger) Printf(format string, args ...interface{}) {}