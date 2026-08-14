package main

import (
	"io"
	"net/http"
	"os/exec"
	"sync"

	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

// ---------------------------------------------------------------------------
// Terminal WebSocket endpoints
//
//   - /ws/terminal/server/:serverId   → host shell scoped to a registered server
//   - /ws/terminal/container/:containerId → podman exec shell into a container
//
// Both authenticate via JWT ?token= and pipe a PTY-less /bin/sh (or the
// container shell) over the socket — same framing as the legacy xterm stream.
// The FE (xterm.js) will connect to these later; only the WS endpoints are
// implemented now.
// ---------------------------------------------------------------------------

// terminalHandler upgrades the connection and spawns the appropriate shell.
// kind is "server" or "container"; id is the URL segment.
func terminalHandler(ctx *fasthttp.RequestCtx, kind, id string) {
	cmd := buildTerminalCmd(kind, id)
	terminalHandlerWithCmd(ctx, cmd)
}

// terminalHandlerWithCmd upgrades the connection and runs the given shell
// command over the WS (used by both the legacy path and the K8s path).
func terminalHandlerWithCmd(ctx *fasthttp.RequestCtx, cmd *exec.Cmd) {
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

	upgrader.Upgrade(ctx, func(conn *websocket.Conn) {
		defer conn.Close()
		runShellOverWS(conn, cmd)
	})
}

// buildTerminalCmd resolves the shell command for a server/container id.
func buildTerminalCmd(kind, id string) *exec.Cmd {
	switch kind {
	case "container":
		// podman exec -it <containerId> /bin/sh (docker fallback)
		if _, err := exec.LookPath("podman"); err == nil {
			return exec.Command("podman", "exec", "-it", id, "/bin/sh")
		}
		if _, err := exec.LookPath("docker"); err == nil {
			return exec.Command("docker", "exec", "-it", id, "/bin/sh")
		}
		return exec.Command("/bin/sh")
	default: // server — host shell
		return exec.Command("/bin/sh")
	}
}

// runShellOverWS wires the subprocess stdio to the WebSocket (shared with the
// legacy streamHandler so both terminal paths behave identically).
func runShellOverWS(conn *websocket.Conn, cmd *exec.Cmd) {
	// Never let a write to a closed conn crash the whole server.
	defer func() {
		if r := recover(); r != nil {
			// client disconnected mid-write — ignore
		}
	}()

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
	writeWS(conn, []byte("\r\n\x1b[33m[process exited]\x1b[0m\r\n"))
	wg.Wait()
}
