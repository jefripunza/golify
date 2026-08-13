package main

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

// RealTimeHub — WebSocket hub for /api/ws/realtime.
// Connected clients (dashboard tabs) receive mutation events whenever
// projects / environments / services / domains / deployments change, so the
// frontend can refetch ONLY when something actually changed (no polling).
type RealTimeHub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]struct{}
}

var hub = &RealTimeHub{clients: make(map[*websocket.Conn]struct{})}

// RealtimeEvent is the JSON payload broadcast to all connected clients.
type RealtimeEvent struct {
	Type   string `json:"type"`   // project | environment | service | domain | deployment | health
	Action string `json:"action"` // created | updated | deleted | finished
	ID     string `json:"id,omitempty"`
	At     int64  `json:"at"` // unix millis
}

// Broadcast sends an event to every connected client.
func (h *RealTimeHub) Broadcast(ev RealtimeEvent) {
	ev.At = time.Now().UnixMilli()
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for conn := range h.clients {
		writeWS(conn, data)
	}
}

// notify is a tiny helper used at every mutation site.
func notify(typ, action, id string) {
	hub.Broadcast(RealtimeEvent{Type: typ, Action: action, ID: id})
}

// realtimeHandler — /api/ws/realtime?token=... upgrades to WS and keeps the
// connection open, pushing events as they happen. Heartbeat pings every 30s.
func realtimeHandler(ctx *fasthttp.RequestCtx) {
	tok := string(ctx.QueryArgs().Peek("token"))
	if tok == "" {
		ctx.SetStatusCode(401)
		ctx.SetBodyString("missing token")
		return
	}
	if _, err := parseToken(tok); err != nil {
		ctx.SetStatusCode(401)
		ctx.SetBodyString("invalid token")
		return
	}

	upgrader := websocket.FastHTTPUpgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(_ *fasthttp.RequestCtx) bool { return true },
	}
	upgrader.Upgrade(ctx, func(conn *websocket.Conn) {
		hub.mu.Lock()
		hub.clients[conn] = struct{}{}
		hub.mu.Unlock()
		defer func() {
			hub.mu.Lock()
			delete(hub.clients, conn)
			hub.mu.Unlock()
			conn.Close()
		}()

		// read loop: consume client messages (ping/pong/close) so the conn
		// stays alive; on any read error the connection is closed.
		done := make(chan struct{})
		go func() {
			defer close(done)
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		}()

		// heartbeat ping every 30s keeps proxies (cloudflared) happy
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				writeWS(conn, []byte(`{"type":"ping"}`))
			}
		}
	})
}
