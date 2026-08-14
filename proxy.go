package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

// proxyDomainGate protects the PROXY ports (everything except the dashboard
// FE port :3000) from serving the Golify login page to arbitrary hostnames.
//
// Only hostnames registered in the domains table — either exactly, or by
// their base domain when the request is a subdomain — that also have a
// service attached may reach the SPA. Anything else gets a plain-text
// notice instead of the login screen:
//
//	not registered            → "please setup your domain on system ..."
//	registered, no service    → "service not found !!!"
func proxyDomainGate(c fiber.Ctx) error {
	// Let API / ACME / assets / WebSocket paths pass through untouched.
	p := c.Path()
	if strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/ws") ||
		strings.HasPrefix(p, "/assets/") || strings.HasPrefix(p, "/.well-known/") {
		return c.Next()
	}

	host := strings.ToLower(c.Hostname())
	// strip port if present (Host header may carry :port on direct access)
	if i := strings.LastIndex(host, ":"); i > 0 && !strings.Contains(host[i:], "]") {
		host = host[:i]
	}
	host = strings.TrimSuffix(host, ".")
	// www.example.com ≡ example.com — normalize before lookup
	host = strings.TrimPrefix(host, "www.")

	if host == "" {
		return plainNotice(c, "please setup your domain on system ...")
	}

	switch checkDomainRegistration(host) {
	case -1:
		return plainNotice(c, "please setup your domain on system ...")
	case 0:
		return plainNotice(c, "service not found !!!")
	}
	// Registered + has a service → reverse-proxy to the container if the
	// service has a mapped port, otherwise serve the SPA.
	if targets := serviceProxyTargets(host, string(c.Protocol())); len(targets) > 0 {
		// force-https redirect target
		if strings.HasPrefix(targets[0], "redirect-https:") {
			dest := strings.TrimPrefix(targets[0], "redirect-https:")
			c.Redirect().Status(fiber.StatusMovedPermanently).To("https://" + dest + c.OriginalURL())
			return nil
		}
		return proxyToBackend(c, targets)
	}
	return c.Next()
}

// plainNotice renders a plain-text body with an HTML content type, as
// requested ("tampilan menjadi text aja di content type html").
func plainNotice(c fiber.Ctx, msg string) error {
	c.Set(fiber.HeaderContentType, "text/html; charset=utf-8")
	return c.Status(fiber.StatusOK).SendString(msg)
}

// fullServiceDomainHost builds the complete hostname for a ServiceDomain:
// <subdomain>.<domain.host>, or bare <domain.host> when subdomain is empty.
// Returns "" when the root domain is missing.
func fullServiceDomainHost(sd ServiceDomain) string {
	if sd.Domain == nil {
		var dom Domain
		if err := db.First(&dom, "id = ?", sd.DomainID).Error; err != nil {
			return ""
		}
		sd.Domain = &dom
	}
	host := sd.Subdomain
	if host != "" {
		host += "."
	}
	return host + sd.Domain.Host
}

// checkDomainRegistration resolves a request hostname against the domains
// table and reports whether it may reach the SPA:
//
//	-1 → hostname (or its base domain) is not registered in domains
//	 0 → registered but no service is attached to it
//	 1 → registered and a service exists → allow through
func checkDomainRegistration(host string) int {
	// A subdomain (3+ labels) is checked against its base domain
	// (last two labels) as well, e.g. test.simtaru.online → simtaru.online.
	candidates := []string{host}
	parts := strings.Split(host, ".")
	if len(parts) >= 3 {
		candidates = append(candidates, strings.Join(parts[len(parts)-2:], "."))
	}

	// 1) must be registered in the domains table
	var n int64
	if err := db.Model(&Domain{}).Where("host IN ?", candidates).Count(&n).Error; err != nil || n == 0 {
		return -1
	}

	// 2) must have a service attached: either the env-scoped Domain whose
	//    environment has at least one Service, OR a ServiceDomain row
	//    (subdomain/domain mapped directly on a service).
	var svc int64
	if err := db.Model(&Service{}).
		Where("environment_id IN (?)",
			db.Model(&Domain{}).Where("host IN ?", candidates).
				Where("environment_id IS NOT NULL").Select("environment_id"),
		).Count(&svc).Error; err != nil || svc == 0 {
		// fallback: service_domains (subdomain mapped on a registered root
		// domain). Build the full host from subdomain + domain.host and match.
		if err := db.Model(&ServiceDomain{}).
			Joins("JOIN domains d ON d.id = service_domains.domain_id").
			Where("CASE WHEN service_domains.subdomain = '' THEN d.host = ? ELSE CONCAT(service_domains.subdomain, '.', d.host) = ? END", host, host).
			Count(&svc).Error; err != nil || svc == 0 {
			return 0
		}
	}
	return 1
}

// serviceProxyTarget resolves the container backend for a request hostname.
// With replicas it returns a list of upstreams (all running replicas) so the
// proxy can load-balance; single replica keeps the old single-target path.
func serviceProxyTargets(host, scheme string) []string {
	candidates := []string{host}
	parts := strings.Split(host, ".")
	if len(parts) >= 3 {
		candidates = append(candidates, strings.Join(parts[len(parts)-2:], "."))
	}
	log.Printf("[proxy] serviceProxyTargets host=%s candidates=%v", host, candidates)
	var sd ServiceDomain
	// match subdomain+root-domain combo: build the full host and compare.
	// (SQLite CONCAT/CASE — cross-check with the registered root domains.)
	if err := db.Preload("Domain").
		Joins("JOIN domains d ON d.id = service_domains.domain_id").
		Where("CASE WHEN service_domains.subdomain = '' THEN d.host IN ? ELSE CONCAT(service_domains.subdomain, '.', d.host) IN ? END", candidates, candidates).
		First(&sd).Error; err != nil {
		return nil
	}
	// force HTTPS: the request must be secure, otherwise redirect.
	if sd.IsForceHTTPS && scheme != "https" {
		return []string{"redirect-https:" + fullServiceDomainHost(sd)}
	}
	var svc Service
	if err := db.First(&svc, "id = ?", sd.ServiceID).Error; err != nil {
		return nil
	}
	// K8s mode: traffic goes through the cluster's Ingress controller.
	// Load balancing is handled by Kubernetes (Service ClusterIP + kube-proxy
	// round robin / Ingress upstreams) — we just forward to the ingress entry
	// point. For kind+podman the only reachable entry is a kubectl
	// port-forward to the ingress-nginx controller (GOLIFY_K8S_INGRESS_ADDR).
	if svc.LoadBalancer == "k8s" {
		addr := os.Getenv("GOLIFY_K8S_INGRESS_ADDR")
		if addr == "" {
			addr = "127.0.0.1:18080" // dev default: kubectl port-forward ingress
		}
		log.Printf("[proxy] %s → K8s ingress %s", host, addr)
		return []string{addr}
	}
	// prefer networks host_port (explicit mapping), else ports[0] host part
	basePort := ""
	for _, n := range svc.Networks {
		if n.HostPort != "" {
			basePort = n.HostPort
			break
		}
	}
	if basePort == "" && len(svc.Ports) > 0 {
		basePort = strings.Split(svc.Ports[0], ":")[0]
	}
	if basePort == "" {
		return nil
	}
	// Single replica → exactly one upstream (legacy behaviour).
	if svc.Replicas <= 1 {
		return []string{"127.0.0.1:" + basePort}
	}
	// Multi-replica → resolve every running container of this service.
	// Containers are named golify-<name> (replica 1) and golify-<name>-N.
	base := "golify-" + strings.ToLower(strings.ReplaceAll(svc.Name, " ", "-"))
	out, err := exec.Command("podman", "ps", "--filter", "name="+base,
		"--format", "{{.Names}}	{{.Ports}}").CombinedOutput()
	if err != nil {
		log.Printf("[proxy] list replicas failed: %v", err)
		return []string{"127.0.0.1:" + basePort}
	}
	var targets []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Split(line, "	")
		if len(fields) < 2 {
			continue
		}
		ports := fields[1]
		// ports format: "127.0.0.1:32701->80/tcp, 127.0.0.1:32702->80/tcp"
		for _, p := range strings.Split(ports, ",") {
			p = strings.TrimSpace(p)
			hostport := strings.Split(p, "->")[0]
			hostport = strings.TrimPrefix(hostport, "127.0.0.1:")
			hostport = strings.TrimSuffix(hostport, "/tcp")
			if hostport != "" && !strings.Contains(hostport, ":") {
				targets = append(targets, "127.0.0.1:"+hostport)
				break
			}
		}
	}
	if len(targets) == 0 {
		return []string{"127.0.0.1:" + basePort}
	}
	return targets
}

// ─── Load balancer state ───────────────────────────────────────────────────
// round-robin counters per service ID (atomic), plus active-connection
// counters per upstream for the least-connection strategy.
var (
	lbRRCounters   = map[string]*atomic.Uint64{}
	lbConnCounters = map[string]*atomic.Int64{} // key: serviceID|upstream
	lbMu           sync.Mutex
)

func lbCounter(key string) *atomic.Uint64 {
	lbMu.Lock()
	defer lbMu.Unlock()
	c, ok := lbRRCounters[key]
	if !ok {
		c = &atomic.Uint64{}
		lbRRCounters[key] = c
	}
	return c
}

func lbConnCounter(key string) *atomic.Int64 {
	lbMu.Lock()
	defer lbMu.Unlock()
	c, ok := lbConnCounters[key]
	if !ok {
		c = &atomic.Int64{}
		lbConnCounters[key] = c
	}
	return c
}

// pickUpstream selects one target from the replica list based on the
// service's load-balancer strategy (round_robin default, least_conn).
func pickUpstream(serviceID, strategy string, targets []string) string {
	if len(targets) == 0 {
		return ""
	}
	if len(targets) == 1 {
		return targets[0]
	}
	switch strategy {
	case "least_conn":
		// pick the upstream with the fewest active connections (tie → first)
		best := targets[0]
		bestN := lbConnCounter(serviceID + "|" + best).Load()
		for _, t := range targets[1:] {
			n := lbConnCounter(serviceID + "|" + t).Load()
			if n < bestN {
				best, bestN = t, n
			}
		}
		lbConnCounter(serviceID + "|" + best).Add(1)
		return best
	default: // round_robin
		n := lbCounter(serviceID).Add(1)
		return targets[(int(n)-1)%len(targets)]
	}
}

// proxyToBackendK8s forwards to the K8s Ingress via net/http, overriding
// the Host header so the Ingress rule (test1.simtaru.online) matches.
// Load balancing happens inside the cluster (Service + kube-proxy).
func proxyToBackendK8s(c fiber.Ctx, target, host string) error {
	log.Printf("[proxy] k8s net/http forward → %s (Host: %s)", target, host)
	upstream := "http://" + target
	path := string(c.Request().URI().RequestURI())
	req, err := http.NewRequestWithContext(c.Context(), string(c.Request().Header.Method()), upstream+path, bytes.NewReader(c.Request().Body()))
	if err != nil {
		return c.Status(fiber.StatusBadGateway).SendString("proxy error: " + err.Error())
	}
	req.Host = host
	// copy headers (skip hop-by-hop)
	c.Request().Header.VisitAll(func(k, v []byte) {
		key := string(k)
		if strings.EqualFold(key, "Host") || strings.EqualFold(key, "Connection") ||
			strings.EqualFold(key, "Upgrade") || strings.EqualFold(key, "Proxy-Connection") {
			return
		}
		req.Header.Set(key, string(v))
	})
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).SendString("proxy error: " + err.Error())
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).SendString("proxy error: " + err.Error())
	}
	c.Response().SetStatusCode(resp.StatusCode)
	c.Response().SetBody(body)
	for k, vs := range resp.Header {
		for _, v := range vs {
			c.Response().Header.Add(k, v)
		}
	}
	return nil
}


func proxyToBackend(c fiber.Ctx, targets []string) error {
	if len(targets) == 0 {
		return c.Status(fiber.StatusBadGateway).SendString("proxy error: no upstream")
	}
	// resolve the strategy for this service from the request host
	// Use the Host HEADER explicitly (not c.Hostname(), which Fiber may
	// derive from the connection address when proxied).
	host := strings.ToLower(string(c.Request().Header.Peek("Host")))
	if i := strings.LastIndex(host, ":"); i > 0 && !strings.Contains(host[i:], "]") {
		host = host[:i]
	}
	host = strings.TrimPrefix(strings.TrimSuffix(host, "."), "www.")
	strategy := "round_robin"
	var svc Service
	// resolve the service through the new subdomain+domain schema
	var sd ServiceDomain
	if err := db.Preload("Domain").
		Joins("JOIN domains d ON d.id = service_domains.domain_id").
		Where("CASE WHEN service_domains.subdomain = '' THEN d.host = ? ELSE CONCAT(service_domains.subdomain, '.', d.host) = ? END", host, host).
		First(&sd).Error; err == nil {
		if err := db.First(&svc, "id = ?", sd.ServiceID).Error; err == nil && svc.LoadBalancer != "" {
			strategy = svc.LoadBalancer
		}
	}
	target := pickUpstream(string(svc.ID), strategy, targets)
	if target == "" {
		return c.Status(fiber.StatusBadGateway).SendString("proxy error: no upstream")
	}

	// count active connection for least_conn
	connKey := string(svc.ID) + "|" + target
	if strategy == "least_conn" {
		lbConnCounter(connKey).Add(1)
		defer lbConnCounter(connKey).Add(-1)
	}

	upstream := "http://" + target
	_ = upstream // (kept for clarity; dialing happens via req.URI().SetHost)
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	// Build a fresh request: method, path+query, headers, body from the
	// incoming request — but a clean URI against the upstream (no redirect
	// loops, no cross-protocol hop).
	c.Request().Header.VisitAll(func(k, v []byte) {
		key := string(k)
		if strings.EqualFold(key, "Host") || strings.EqualFold(key, "Connection") ||
			strings.EqualFold(key, "Upgrade") || strings.EqualFold(key, "Proxy-Connection") {
			return
		}
		req.Header.SetBytesKV(k, v)
	})
	req.Header.SetMethodBytes(c.Request().Header.Method())
	// K8s path: use net/http with an explicit Host override — fasthttp's
	// URI/Host handling is too easy to get wrong for Ingress host routing.
	if strategy == "k8s" {
		return proxyToBackendK8s(c, target, host)
	}
	// Use an absolute URI so fasthttp dials the upstream, then override the
	// Host header on the wire (K8s Ingress routes on the original Host).
	req.SetRequestURI(upstream + string(c.Request().URI().RequestURI()))
	req.Header.SetHost(target)
	if body := c.Request().Body(); len(body) > 0 {
		req.SetBody(body)
	}

	client := &fasthttp.Client{
		NoDefaultUserAgentHeader: true,
		// never follow redirects — mirror them to the caller
		MaxResponseBodySize: 64 << 20,
	}
	if err := client.Do(req, resp); err != nil {
		return c.Status(fiber.StatusBadGateway).SendString("proxy error: " + err.Error())
	}

	c.Response().SetStatusCode(resp.StatusCode())
	c.Response().SetBody(resp.Body())
	resp.Header.VisitAll(func(k, v []byte) {
		c.Response().Header.SetBytesKV(k, v)
	})
	return nil
}
