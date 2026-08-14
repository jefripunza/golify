// Package main is the Gotify server entrypoint.
//
// Stack:
//   - Fiber v3 (HTTP framework)
//   - GORM (ORM) with the standard mattn/go-sqlite3 driver
//   - golang-jwt/jwt v5 for auth tokens
//
// Single binary; the Vue 3 frontend is embedded via go:embed.
package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/tls"
	"database/sql"
	"embed"
	"errors"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/etag"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

//go:embed all:web/dist
var webDist embed.FS

const (
	portBE    = ":20000" // BE Go: API + WS dashboard (WS prefix /api/ws/*)
	portHTTP  = ":20001" // proxy HTTP + ACME → FE :20003
	portHTTPS = ":20002" // proxy HTTPS → FE :20003
	portFE    = ":20003" // FE Vue Vite (dev server, HMR ws sendiri)
	portDual  = ":8080"  // proxy all-in-one (HTTP+HTTPS+ACME) → FE :20003
	dataDir   = "data"
)

var (
	db     *gorm.DB
	jwtKey = loadOrCreateJwtKey()
)

// jwtKeyFile persists the JWT signing key so tokens survive restarts.
// Without this, every backend restart invalidates all sessions (random key)
// → WebSocket auth 401 → dashboard shows Offline with empty metrics.
const jwtKeyFile = "data/jwt.key"

func loadOrCreateJwtKey() []byte {
	if b, err := os.ReadFile(jwtKeyFile); err == nil && len(b) >= 32 {
		return b
	}
	key := mustRandomKey(32)
	if err := os.MkdirAll(filepath.Dir(jwtKeyFile), 0o755); err == nil {
		if err := os.WriteFile(jwtKeyFile, key, 0o600); err != nil {
			log.Printf("warn: cannot persist jwt key to %s: %v", jwtKeyFile, err)
		}
	}
	return key
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func mustRandomKey(n int) []byte {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("rand: %v", err)
	}
	return b
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	// Load .env (godotenv) BEFORE reading any env var — port/bind overrides
	// live in .env (see .env.example). Real env vars take precedence.
	if err := godotenv.Load(); err == nil {
		log.Printf("env: loaded .env")
	} else {
		log.Printf("env: no .env found (%v) — using defaults", err)
	}

	// on-disk layout
	if err := os.MkdirAll(filepath.Join(dataDir, "ssl", "letsencrypt"), 0o755); err != nil {
		log.Fatalf("mkdir data/ssl/letsencrypt: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "ssl", "custom"), 0o755); err != nil {
		log.Fatalf("mkdir data/ssl/custom: %v", err)
	}

	// GORM + SQLite (WAL + foreign keys)
	dbPath := filepath.Join(dataDir, "golify.db")
	dsn := dbPath + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	var err error
	db, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		log.Fatalf("gorm open: %v", err)
	}
	// v0.2 service_domains redesign: old rows used `host`/`port` columns with a
	// unique service_id+host index. New model uses `domain_id` + `subdomain` +
	// `is_force_https` (unique on subdomain+domain_id). SQLite cannot add NOT
	// NULL columns to a non-empty table, so drop the old table and let
	// AutoMigrate recreate it with the new schema.
	if db.Migrator().HasTable("service_domains") && !db.Migrator().HasColumn("service_domains", "domain_id") {
		log.Println("[migrate] dropping legacy service_domains (host/port schema) → recreated by AutoMigrate")
		if err := db.Migrator().DropTable(&ServiceDomain{}); err != nil {
			log.Fatalf("drop legacy service_domains: %v", err)
		}
	}
	if err := db.AutoMigrate(&User{}, &Project{}, &Environment{}, &Service{}, &ServiceDomain{}, &ServiceNetwork{}, &Deployment{}, &Domain{}, &Server{}, &Source{}, &S3Storage{}, &SharedVariable{}, &Key{}, &ApiKey{}, &Team{}, &TeamMember{}); err != nil {
		log.Fatalf("automigrate: %v", err)
	}
	log.Printf("sqlite ready at %s (GORM) — fresh DB, no seeder (onboarding-first)", dbPath)

	// system analytics worker — 1 Hz host metrics for /ws/analytic
	startSystemWorker()
	log.Println("system analytics worker started (1 Hz)")

	// :20000 — BE Go: SPA (embed) + API + WS dashboard (WS prefix /api/ws/*).
	// Inilah satu-satunya backend dashboard. Real proxy (20001/20002/8080)
	// adalah server standalone — TIDAK forward ke sini. FE Vite (20003)
	// proxy /api ke sini (vite proxy).
	beApp := rootSPA("be")
	bindBE := envOr("GOTIFY_BE", portBE)
	lnBE, err := net.Listen("tcp", bindBE)
	if err != nil {
		log.Fatalf("listen %s: %v", bindBE, err)
	}
	go func() {
		log.Printf("Backend listening on %s (SPA + API + WS /api/ws/*)", bindBE)
		if err := serveUnified(lnBE, beApp.Handler()); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("backend server: %v", err)
		}
	}()

	// :20001 — real proxy HTTP + ACME (standalone, no API)
	httpProxy := newProxyApp("http-proxy", true)
	bindHTTP := envOr("GOTIFY_HTTP", portHTTP)
	lnHTTP, err := net.Listen("tcp", bindHTTP)
	if err != nil {
		log.Fatalf("listen %s: %v", bindHTTP, err)
	}
	go func() {
		log.Printf("HTTP real-proxy listening on %s (gate) — standalone SPA+ACME (no API)", bindHTTP)
		if err := httpProxy.Listener(lnHTTP); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("http proxy: %v", err)
		}
	}()

	// :20002 — real proxy HTTPS (standalone, no API)
	httpsProxy := newProxyApp("https-proxy", true)
	bindHTTPS := envOr("GOTIFY_HTTPS", portHTTPS)
	lnHTTPS, err := tls.Listen("tcp", bindHTTPS, buildTLSConfig())
	if err != nil {
		log.Fatalf("listen %s: %v", bindHTTPS, err)
	}
	go func() {
		log.Printf("HTTPS real-proxy listening on %s (gate) — standalone SPA+ACME+TLS (no API)", bindHTTPS)
		if err := httpsProxy.Listener(lnHTTPS); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("https proxy: %v", err)
		}
	}()

	// :8080 — real proxy all-in-one (HTTP + HTTPS + ACME on ONE port, standalone, no API).
	// Cloudflare ingress rules for simtaru.online / wajadi.online point here.
	dualProxy := newProxyApp("dual-proxy", true)
	bindDual := envOr("GOTIFY_DUAL", portDual)
	lnDual, err := net.Listen("tcp", bindDual)
	if err != nil {
		log.Fatalf("listen dual %s: %v", bindDual, err)
	}
	go func() {
		log.Printf("DUAL real-proxy listening on %s (gate) — standalone SPA+ACME+TLS (no API)", bindDual)
		dual := &dualListener{Listener: lnDual, tlsCfg: buildTLSConfig()}
		if err := serveUnified(dual, dualProxy.Handler()); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("dual proxy: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("shutdown signal received")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, app := range []*fiber.App{beApp, httpProxy, httpsProxy, dualProxy} {
		_ = app.ShutdownWithContext(ctx)
	}
	log.Println("bye")
}

// newProxyApp builds a REAL proxy: a standalone server that serves the SPA
// (static build, embedded) + ACME challenges + TLS + domain gate. It does
// NOT run the dashboard API (that lives ONLY on the BE :20000) and does NOT
// forward anywhere — it is its own backend, separate from the dashboard BE.
func newProxyApp(label string, gate bool) *fiber.App {
	app := newFiber(label)
	if gate {
		app.Use(proxyDomainGate)
	}
	// real ACME: certbot --webroot writes challenge files into
	// data/ssl/letsencrypt/.acme/<token>; serve them straight from disk.
	app.Get("/.well-known/acme-challenge/:token", acmeChallenge)
	// Dashboard API is NOT served on proxy ports (haram) — explicit 404 so
	// it never falls through to the SPA.
	app.All("/api/*", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).SendString("api not served on this proxy port")
	})
	mountSPA(app, label)
	return app
}

// ----- middleware stack ---------------------------------------------------

func newFiber(label string) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "golify-" + label,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	})
	app.Use(requestid.New())
	app.Use(recover.New())
	app.Use(etag.New())
	app.Use(compress.New(compress.Config{Level: compress.LevelBestSpeed}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Authorization", "X-Requested-With"},
	}))
	app.Use(limiter.New(limiter.Config{
		Max:        200,
		Expiration: 1 * time.Minute,
	}))
	app.Use(logger.New(logger.Config{Format: "[${time}] ${ip} ${status} ${method} ${path}\n"}))
	return app
}

// ----- ACME http-01 solver -------------------------------------------------

func acmeChallenge(c fiber.Ctx) error {
	token := c.Params("token")
	// certbot --webroot writes challenge files into the webroot dir;
	// we expect them at /app/data/ssl/letsencrypt/.acme/<token>
	fp := filepath.Join(dataDir, "ssl", "letsencrypt", ".acme", token)
	b, err := os.ReadFile(fp)
	if err != nil {
		return c.Status(404).SendString("challenge not found")
	}
	c.Set(fiber.HeaderContentType, "text/plain")
	return c.Send(b)
}

// ----- SPA mounting --------------------------------------------------------

// rootSPA builds the BE app: full SPA (embed) + API + WS handler (via
// serveUnified). It is used ONLY for the BE port :20000. Real proxy ports
// use newProxyApp (pure forwarder) instead.
func rootSPA(label string) *fiber.App {
	app := newFiber(label)
	api := app.Group("/api")
	registerAPI(api)
	mountSPA(app, label)
	return app
}

func mountSPA(app *fiber.App, label string) {
	dist, err := fs.Sub(webDist, "web/dist")
	if err != nil {
		log.Fatalf("[%s] embed sub web/dist: %v", label, err)
	}
	app.Get("/", serveIndex(dist))
	app.Get("/*", spaFallback(dist))
	app.Use("/", staticFromFS(dist))
}

func serveIndex(dist fs.FS) fiber.Handler {
	return func(c fiber.Ctx) error {
		b, err := fs.ReadFile(dist, "index.html")
		if err != nil {
			return c.Status(500).SendString("SPA index.html missing in embed")
		}
		c.Set(fiber.HeaderContentType, "text/html; charset=utf-8")
		return c.Send(b)
	}
}

func spaFallback(dist fs.FS) fiber.Handler {
	return func(c fiber.Ctx) error {
		path := c.Path()
		rel := strings.TrimPrefix(path, "/")
		if rel == "" {
			rel = "index.html"
		}
		if _, err := fs.Stat(dist, rel); err == nil {
			return c.Next()
		}
		// A missing file under /assets/ (or any .js/.css/.mjs) must 404, NOT
		// fall back to index.html — otherwise the browser gets text/html for a
		// script/stylesheet request and throws "not a valid JavaScript MIME
		// type", breaking lazy-loaded chunks after a deploy (old index.html
		// references chunks that no longer exist).
		if strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".css") || strings.HasSuffix(path, ".mjs") || strings.HasSuffix(path, ".map") || strings.HasSuffix(path, ".woff") || strings.HasSuffix(path, ".woff2") || strings.HasSuffix(path, ".png") || strings.HasSuffix(path, ".svg") || strings.HasSuffix(path, ".ico") || strings.HasSuffix(path, ".json") || strings.HasSuffix(path, ".webmanifest") {
			return c.Status(404).SendString("not found")
		}
		b, err := fs.ReadFile(dist, "index.html")
		if err != nil {
			return c.Status(500).SendString("SPA index.html missing in embed")
		}
		c.Set(fiber.HeaderContentType, "text/html; charset=utf-8")
		return c.Send(b)
	}
}

func staticFromFS(fsys fs.FS) fiber.Handler {
	fileServer := http.FileServer(http.FS(fsys))
	return func(c fiber.Ctx) error {
		path := c.Path()
		rel := strings.TrimPrefix(path, "/")
		if rel == "" {
			return c.Next()
		}
		req, err := http.NewRequest(http.MethodGet, "/"+rel, nil)
		if err != nil {
			return err
		}
		rec := &recWriter{header: http.Header{}}
		fileServer.ServeHTTP(rec, req)
		if rec.status == 0 {
			rec.status = 200
		}
		for k, vs := range rec.header {
			for _, v := range vs {
				c.Set(k, v)
			}
		}
		// no-cache: assets are content-hashed (immutable per build), but the
		// Cloudflare tunnel edge can otherwise serve a STALE copy of a chunk
		// whose URL name collides across builds (Vite keeps the same hash when
		// only an import specifier changes). Revalidating every request costs
		// nothing here and eliminates stale-chunk mixing entirely.
		c.Set(fiber.HeaderCacheControl, "no-cache")
		c.Status(rec.status)
		return c.Send(rec.body)
	}
}

type recWriter struct {
	header http.Header
	body   []byte
	status int
}

func (r *recWriter) Header() http.Header         { return r.header }
func (r *recWriter) Write(b []byte) (int, error) { r.body = append(r.body, b...); return len(b), nil }
func (r *recWriter) WriteHeader(s int)           { r.status = s }

// ----- TLS ------------------------------------------------------------------

// dualListener accepts both plain HTTP and TLS on the same port. It peeks
// the first byte of each connection: 0x16 (TLS handshake record) → wrap the
// conn in tls.Server with SNI-based cert lookup; anything else → plain HTTP.
// This lets one Cloudflare ingress rule serve both http:// and https://.
type dualListener struct {
	net.Listener
	tlsCfg *tls.Config
}

func (dl *dualListener) Accept() (net.Conn, error) {
	raw, err := dl.Listener.Accept()
	if err != nil {
		return nil, err
	}
	br := bufio.NewReader(raw)
	first, err := br.Peek(1)
	if err != nil {
		raw.Close()
		return nil, err
	}
	// TLS record content type for a handshake is 0x16.
	if first[0] == 0x16 {
		return tls.Server(&peekConn{Conn: raw, r: br}, dl.tlsCfg), nil
	}
	return &peekConn{Conn: raw, r: br}, nil
}

// peekConn is a net.Conn whose Read drains the buffered reader first.
type peekConn struct {
	net.Conn
	r *bufio.Reader
}

func (pc *peekConn) Read(b []byte) (int, error) { return pc.r.Read(b) }

func buildTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			sn := hello.ServerName
			if sn == "" {
				return nil, nil
			}
			candidates := []struct{ cert, key string }{
				{
					filepath.Join(dataDir, "ssl", "custom", sn+".crt"),
					filepath.Join(dataDir, "ssl", "custom", sn+".key"),
				},
				{
					filepath.Join(dataDir, "ssl", "letsencrypt", sn, "fullchain.pem"),
					filepath.Join(dataDir, "ssl", "letsencrypt", sn, "privkey.pem"),
				},
			}
			for _, c := range candidates {
				if cert, err := tls.LoadX509KeyPair(c.cert, c.key); err == nil {
					return &cert, nil
				}
			}
			return nil, nil
		},
	}
}

// ----- JWT helpers ----------------------------------------------------------

type Claims struct {
	UserID   UUID   `json:"uid"`
	Username string `json:"usr"`
	Admin    bool   `json:"adm"`
	jwt.RegisteredClaims
}

func issueToken(u User) (string, error) {
	claims := Claims{
		UserID:   u.ID,
		Username: u.Username,
		Admin:    u.Admin,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			Issuer:    "golify",
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(jwtKey)
}

func parseToken(s string) (*Claims, error) {
	tok, err := jwt.ParseWithClaims(s, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtKey, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := tok.Claims.(*Claims); ok && tok.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}

// requireAuth is a Fiber middleware that requires a valid Bearer JWT.
func requireAuth(c fiber.Ctx) error {
	h := string(c.Request().Header.Peek(fiber.HeaderAuthorization))
	if !strings.HasPrefix(h, "Bearer ") {
		return c.Status(401).JSON(fiber.Map{"error": "missing bearer token"})
	}
	claims, err := parseToken(strings.TrimPrefix(h, "Bearer "))
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": err.Error()})
	}
	c.Locals("claims", claims)
	return c.Next()
}

// requireAdmin additionally enforces admin role.
func requireAdmin(c fiber.Ctx) error {
	claims, ok := c.Locals("claims").(*Claims)
	if !ok || claims == nil || !claims.Admin {
		return c.Status(403).JSON(fiber.Map{"error": "admin only"})
	}
	return c.Next()
}

// newID returns a UUID v7 string (time-ordered primary key for all tables).
func newID() UUID {
	id, err := uuid.NewV7()
	if err != nil {
		// extremely unlikely — fall back to a random UUID
		return uuid.NewString()
	}
	return id.String()
}

// shortID returns the first 8 chars of a UUID (safe, unique enough for
// container names / human-readable references).
func shortID(id UUID) string {
	if len(id) >= 8 {
		return id[:8]
	}
	return id
}

// silence unused-import warnings if sql import drifts later
var _ = sql.ErrNoRows
