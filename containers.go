package main

import (
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
)

// Container count for the dashboard "Total Container" card.
//
// Runs `podman ps -aq` (fallback `docker ps -aq`) and counts lines. The
// result is cached for 5 seconds so we don't spawn a process per request.

var (
	ctrMu       sync.Mutex
	ctrCount    int
	ctrFetched  time.Time
	ctrRuntime  string // "podman" | "docker" | ""
)

const ctrCacheTTL = 5 * time.Second

func containersHandler(c fiber.Ctx) error {
	count, runtime := containerCount()
	return c.JSON(fiber.Map{
		"count":   count,
		"runtime": runtime,
	})
}

// containerCount returns (count, runtime) — cached up to ctrCacheTTL.
func containerCount() (int, string) {
	ctrMu.Lock()
	defer ctrMu.Unlock()

	if time.Since(ctrFetched) < ctrCacheTTL {
		return ctrCount, ctrRuntime
	}

	for _, rt := range []string{"podman", "docker"} {
		if _, err := exec.LookPath(rt); err != nil {
			continue
		}
		out, err := exec.Command(rt, "ps", "-aq").Output()
		if err != nil {
			continue
		}
		lines := strings.Fields(string(out))
		ctrCount = len(lines)
		ctrRuntime = rt
		ctrFetched = time.Now()
		return ctrCount, ctrRuntime
	}

	ctrCount = 0
	ctrRuntime = ""
	ctrFetched = time.Now()
	return 0, ""
}
