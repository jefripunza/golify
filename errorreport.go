package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
)

// Error reporting — the frontend ErrorBoundary posts client-side errors to
// this endpoint; we validate the payload and forward it to the central
// error reporter at error.sawang.tech (which delivers to the Sawang Tech
// Telegram thread). Public on purpose: errors can happen before auth.

const errorReportURL = "https://error.sawang.tech/api/report/error"

// Dedupe: identical reports (same app+title+stack) are only forwarded once
// per window. Guards against the FE boundary firing multiple times (layout
// + router re-mounts) without spamming the error reporter.
var reportMu sync.Mutex
var recentReports = map[string]time.Time{}
const reportDedupeWindow = 60 * time.Second

type errorReport struct {
	AppName string `json:"app_name"`
	AppURL  string `json:"app_url"`
	Title   string `json:"title"`
	Stack   string `json:"stack"`
}

func reportErrorHandler(c fiber.Ctx) error {
	var body errorReport
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"ok": false, "error": "invalid json body"})
	}
	if body.AppName == "" || body.AppURL == "" || body.Title == "" || body.Stack == "" {
		return c.Status(400).JSON(fiber.Map{
			"ok":    false,
			"error": "app_name, app_url, title, stack are required",
		})
	}

	// dedupe identical reports within the window
	key := body.AppName + "\x00" + body.Title + "\x00" + body.Stack
	reportMu.Lock()
	if last, ok := recentReports[key]; ok && time.Since(last) < reportDedupeWindow {
		reportMu.Unlock()
		return c.JSON(fiber.Map{"ok": true, "delivered": true, "duplicate": true})
	}
	recentReports[key] = time.Now()
	reportMu.Unlock()

	payload, err := json.Marshal(body)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"ok": false, "error": err.Error()})
	}

	req, err := http.NewRequest(http.MethodPost, errorReportURL, bytes.NewReader(payload))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"ok": false, "error": err.Error()})
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"ok": false, "error": "error reporter unreachable: " + err.Error()})
	}
	defer resp.Body.Close()

	// Mirror whatever the upstream returned ({"ok":true,"delivered":true}).
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return c.Status(resp.StatusCode).JSON(fiber.Map{"ok": false, "error": "bad upstream response"})
	}
	return c.Status(resp.StatusCode).JSON(out)
}
