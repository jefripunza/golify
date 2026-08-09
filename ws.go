package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

// WebSocket terminal exec — a separate native fasthttp server:
//   1. authenticates via JWT in the `?token=` query string
//   2. resolves the requested service to a podman container (or host shell)
//   3. spawns `podman exec -it <ctr> /bin/sh` (fallback host /bin/sh) with
//      stdin/stdout piped to the WS connection
//
// Runs on its own port (GOTIFY_WS env, default :20004) because Fiber v3 has
// no first-class WebSocket support yet.

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
			conn.WriteMessage(websocket.TextMessage, []byte("\r\n\x1b[31mfailed: "+err.Error()+"\x1b[0m\r\n"))
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
		conn.WriteMessage(websocket.TextMessage, []byte("\r\n\x1b[33m[process exited]\x1b[0m\r\n"))
		wg.Wait()
	})
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
	mu   sync.Mutex
}

func (w *wsWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.conn.WriteMessage(websocket.TextMessage, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

// startWSServer boots the WS listener. Blocks until ctx is cancelled.
func startWSServer(ctx context.Context) error {
	port := getenv("GOTIFY_WS", ":20004")
	if !strings.HasPrefix(port, ":") && !strings.HasPrefix(port, "/") {
		port = ":" + port
	}
	srv := &fasthttp.Server{
		Handler: func(c *fasthttp.RequestCtx) {
			path := string(c.Path())
			if path == "/ws/health" {
				c.SetStatusCode(200)
				c.SetBodyString(`{"app":"golify-ws","status":"ok"}`)
				return
			}
			streamHandler(c)
		},
		Logger: discardLogger{},
	}
	go func() {
		<-ctx.Done()
		srv.Shutdown()
	}()
	return srv.ListenAndServe(port)
}

func getenv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// discardLogger silences fasthttp access logs.
type discardLogger struct{}

func (discardLogger) Printf(format string, args ...interface{}) {}