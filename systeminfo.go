package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/valyala/fasthttp"
)

// ---------------------------------------------------------------------------
// System analytics worker
//
// A single background goroutine collects host metrics every second (gopsutil)
// and stores the latest snapshot in a mutex-protected variable. Every
// connected /ws/analytic client receives that snapshot (no per-client work).
// ---------------------------------------------------------------------------

// systemInfoPayload is the JSON frame sent to /ws/analytic clients.
type systemInfoPayload struct {
	CPU    float64    `json:"cpu"`
	Mem    systemMem  `json:"mem"`
	Disk   systemDisk `json:"disk"`
	Net    systemNet  `json:"net"`
	Uptime uint64     `json:"uptime"`
	Load   [3]float64 `json:"load"`
	Host   string     `json:"host"`
	OS     string     `json:"os"`
}

type systemMem struct {
	Used  float64 `json:"used"`  // GB
	Total float64 `json:"total"` // GB
}

type systemDisk struct {
	Used  float64 `json:"used"`  // GB
	Total float64 `json:"total"` // GB
}

type systemNet struct {
	RX uint64 `json:"rx"` // bytes
	TX uint64 `json:"tx"` // bytes
}

var (
	// latest snapshot + guard; updated by the worker, read by WS clients.
	sysMu       sync.RWMutex
	lastSysInfo systemInfoPayload
)

// startSystemWorker launches the 1-second analytics loop.
func startSystemWorker() {
	go func() {
		for {
			p, err := collectSystemInfo()
			if err == nil {
				sysMu.Lock()
				lastSysInfo = *p
				sysMu.Unlock()
			}
			time.Sleep(1 * time.Second)
		}
	}()
}

// currentSystemInfo returns the latest collected snapshot.
func currentSystemInfo() systemInfoPayload {
	sysMu.RLock()
	defer sysMu.RUnlock()
	return lastSysInfo
}

func collectSystemInfo() (*systemInfoPayload, error) {
	p := &systemInfoPayload{}

	// CPU — overall percent across all cores
	if percs, err := cpu.Percent(0, false); err == nil && len(percs) > 0 {
		p.CPU = percs[0]
	}

	// Memory
	if vm, err := mem.VirtualMemory(); err == nil {
		p.Mem.Used = float64(vm.Used) / (1024 * 1024 * 1024)
		p.Mem.Total = float64(vm.Total) / (1024 * 1024 * 1024)
	}

	// Disk — root partition
	if du, err := disk.Usage("/"); err == nil {
		p.Disk.Used = float64(du.Used) / (1024 * 1024 * 1024)
		p.Disk.Total = float64(du.Total) / (1024 * 1024 * 1024)
	}

	// Network — cumulative counters (bytes since boot)
	if counters, err := net.IOCounters(false); err == nil && len(counters) > 0 {
		p.Net.RX = counters[0].BytesRecv
		p.Net.TX = counters[0].BytesSent
	}

	// Host info
	if hi, err := host.Info(); err == nil {
		p.Uptime = hi.Uptime
		p.Host = hi.Hostname
		p.OS = hi.Platform + " " + hi.PlatformVersion
	}

	// Load average
	if la, err := load.Avg(); err == nil {
		p.Load = [3]float64{la.Load1, la.Load5, la.Load15}
	}

	return p, nil
}

// ---------------------------------------------------------------------------
// /ws/analytic — stream system metrics to dashboard clients
//
// The dashboard opens this socket on mount and closes it on unmount, so a
// connection's lifetime == time spent on the dashboard page. Frames are the
// latest snapshot from the worker (1 Hz).
// ---------------------------------------------------------------------------

func analyticHandler(ctx *fasthttp.RequestCtx) {
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
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		// send first frame immediately
		if !sendAnalyticFrame(conn) {
			return
		}

		for range ticker.C {
			if !sendAnalyticFrame(conn) {
				return
			}
		}
	})
}

func sendAnalyticFrame(conn *websocket.Conn) bool {
	snap := currentSystemInfo()
	b, err := json.Marshal(snap)
	if err != nil {
		return false
	}
	if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
		return false
	}
	return true
}
