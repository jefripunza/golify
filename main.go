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
	"context"
	"crypto/rand"
	"crypto/tls"
	"database/sql"
	"embed"
	"encoding/hex"
	"errors"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/etag"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/golang-jwt/jwt/v5"
	"github.com/glebarez/sqlite"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm"
)

//go:embed all:web/dist
var webDist embed.FS

const (
	portHTTP  = ":80"
	portHTTPS = ":443"
	portFE    = 3000
	dataDir   = "data"
)

var (
	db     *gorm.DB
	jwtKey = mustRandomKey(32)
)

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
	if err := db.AutoMigrate(&Message{}, &Application{}, &Client{}, &User{}, &Project{}, &Environment{}, &Service{}, &Domain{}, &Server{}, &Source{}, &S3Storage{}, &SharedVariable{}, &Key{}, &ApiKey{}, &McpEndpoint{}, &Team{}, &TeamMember{}); err != nil {
		log.Fatalf("automigrate: %v", err)
	}
	seedIfEmpty(db)
	log.Printf("sqlite ready at %s (GORM)", dbPath)

	// system analytics worker — 1 Hz host metrics for /ws/analytic
	startSystemWorker()
	log.Println("system analytics worker started (1 Hz)")

	// :80 — HTTP, ACME solver, API
	httpApp := newHTTPApp()
	bindHTTP := envOr("GOTIFY_HTTP", portHTTP)
	ln80, err := net.Listen("tcp", bindHTTP)
	if err != nil {
		log.Fatalf("listen :80: %v", err)
	}
	go func() {
		log.Printf("HTTP listening on %s (ACME solver + API)", bindHTTP)
		if err := httpApp.Listener(ln80); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("http server: %v", err)
		}
	}()

	// :443 — HTTPS, full SPA + API
	httpsApp := newHTTPSApp()
	bindHTTPS := envOr("GOTIFY_HTTPS", portHTTPS)
	ln443, err := tls.Listen("tcp", bindHTTPS, buildTLSConfig())
	if err != nil {
		log.Fatalf("listen :443: %v", err)
	}
	go func() {
		log.Printf("HTTPS listening on %s (SPA + API)", bindHTTPS)
		if err := httpsApp.Listener(ln443); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("https server: %v", err)
		}
	}()

	// :3000 — dashboard SPA + WebSocket (unified single port)
	dashApp := rootSPA("dashboard")
	bindFE := envOr("GOTIFY_FE", ":"+strconv.Itoa(portFE))
	lnFE, err := net.Listen("tcp", bindFE)
	if err != nil {
		log.Fatalf("listen :%d: %v", portFE, err)
	}
	go func() {
		log.Printf("Dashboard listening on %s (SPA + WS)", bindFE)
		if err := serveUnified(lnFE, dashApp.Handler()); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("dashboard server: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("shutdown signal received")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, app := range []*fiber.App{httpApp, httpsApp, dashApp} {
		_ = app.ShutdownWithContext(ctx)
	}
	log.Println("bye")
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

func newHTTPApp() *fiber.App {
	app := newFiber("http")
	app.Get("/.well-known/acme-challenge/:token", acmeChallenge)
	api := app.Group("/api")
	registerAPI(api)
	return app
}

func newHTTPSApp() *fiber.App {
	app := newFiber("https")
	api := app.Group("/api")
	registerAPI(api)
	mountSPA(app, "https")
	return app
}

func rootSPA(label string) *fiber.App {
	app := newFiber(label)
	mountSPA(app, label)
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
	UserID   uint   `json:"uid"`
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

// randomToken returns a URL-safe hex string of n random bytes.
func randomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

// silence unused-import warnings if sql import drifts later
var _ = sql.ErrNoRows
