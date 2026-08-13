package main

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	gnet "github.com/shirou/gopsutil/v3/net"
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
	CPU        float64    `json:"cpu"`
	Cores      int        `json:"cores"` // total logical CPUs
	CPUPerCore []float64  `json:"cpuPerCore"` // per-logical-core percent
	Mem        systemMem  `json:"mem"`
	Disk       systemDisk `json:"disk"`
	Net        systemNet  `json:"net"`
	Uptime     uint64     `json:"uptime"`
	Load       [3]float64 `json:"load"`
	Host       string     `json:"host"`
	OS         string     `json:"os"`
	IPLocal    string     `json:"ipLocal"`  // primary ethernet IPv4
	IPPublic   string     `json:"ipPublic"` // public IP (detected via api.ipify.org)
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
			refreshPublicIP() // cached 60s; updates lastSysInfo.IPPublic
			p, err := collectSystemInfo()
			if err == nil {
				publishPublicIP(p)
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

	// CPU — overall percent across all cores + logical core count + per-core
	if percs, err := cpu.Percent(0, false); err == nil && len(percs) > 0 {
		p.CPU = percs[0]
	}
	if n, err := cpu.Counts(true); err == nil {
		p.Cores = n
	}
	// per-logical-core percentages (same tick as the aggregate)
	if perCore, err := cpu.Percent(0, true); err == nil {
		p.CPUPerCore = perCore
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
	if counters, err := gnet.IOCounters(false); err == nil && len(counters) > 0 {
		p.Net.RX = counters[0].BytesRecv
		p.Net.TX = counters[0].BytesSent
	}

	// Host info
	if hi, err := host.Info(); err == nil {
		p.Uptime = hi.Uptime
		p.Host = hi.Hostname
		p.OS = hi.Platform + " " + hi.PlatformVersion
	}

	// Local IPv4 — primary ethernet (skip loopback/virtual bridges)
	p.IPLocal = localIPv4()

	// Load average
	if la, err := load.Avg(); err == nil {
		p.Load = [3]float64{la.Load1, la.Load5, la.Load15}
	}

	return p, nil
}

// localIPv4 returns the first non-loopback IPv4 (prefer physical ethernet).
func localIPv4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := ifc.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			if ip4 := ip.To4(); ip4 != nil {
				return ip4.String()
			}
		}
	}
	return ""
}

// refreshPublicIP updates lastSysInfo.IPPublic (best-effort, cached 60s).
// Fails silently — the dashboard just keeps the previous value.
var (
	pubIPMu     sync.Mutex
	lastPubIP   string
	lastPubIPAt time.Time
)

func refreshPublicIP() {
	pubIPMu.Lock()
	defer pubIPMu.Unlock()
	if time.Since(lastPubIPAt) < 60*time.Second {
		return // already fresh
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("https://api.ipify.org")
	if err != nil {
		return
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 64))
	ip := strings.TrimSpace(string(b))
	if net.ParseIP(ip) != nil {
		lastPubIP = ip
		lastPubIPAt = time.Now()
	}
}

// publishPublicIP copies the cached public IP into the snapshot.
func publishPublicIP(p *systemInfoPayload) {
	pubIPMu.Lock()
	p.IPPublic = lastPubIP
	pubIPMu.Unlock()
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
	if err := writeWS(conn, b); err != nil {
		return false
	}
	return true
}
