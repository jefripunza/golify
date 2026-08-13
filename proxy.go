package main

import (
	"log"
	"strings"

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
	if target := serviceProxyTarget(host); target != "" {
		return proxyToBackend(c, target)
	}
	return c.Next()
}

// plainNotice renders a plain-text body with an HTML content type, as
// requested ("tampilan menjadi text aja di content type html").
func plainNotice(c fiber.Ctx, msg string) error {
	c.Set(fiber.HeaderContentType, "text/html; charset=utf-8")
	return c.Status(fiber.StatusOK).SendString(msg)
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
		// fallback: service_domains (host attached straight to a service)
		if err := db.Model(&Service{}).
			Joins("JOIN service_domains sd ON sd.service_id = services.id").
			Where("sd.host IN ?", candidates).
			Count(&svc).Error; err != nil || svc == 0 {
			return 0
		}
	}
	return 1
}

// serviceProxyTarget resolves the container backend for a request hostname.
// It looks up the service_domains row for the host, takes the service's
// mapped host port (ports[0] "hostport:80" or its networks), and returns
// "127.0.0.1:<port>" — empty when the service has no mapped port yet.
func serviceProxyTarget(host string) string {
	candidates := []string{host}
	parts := strings.Split(host, ".")
	if len(parts) >= 3 {
		candidates = append(candidates, strings.Join(parts[len(parts)-2:], "."))
	}
	log.Printf("[proxy] serviceProxyTarget host=%s candidates=%v", host, candidates)
	var sd ServiceDomain
	if err := db.Where("host IN ?", candidates).First(&sd).Error; err != nil {
		return ""
	}
	var svc Service
	if err := db.First(&svc, "id = ?", sd.ServiceID).Error; err != nil {
		return ""
	}
	// prefer networks host_port (explicit mapping), else ports[0] host part
	for _, n := range svc.Networks {
		if n.HostPort != "" {
			return "127.0.0.1:" + n.HostPort
		}
	}
	if len(svc.Ports) > 0 {
		hp := strings.Split(svc.Ports[0], ":")[0]
		if hp != "" {
			return "127.0.0.1:" + hp
		}
	}
	return ""
}

// proxyToBackend forwards the request to an upstream (127.0.0.1:<port>)
// using fasthttp's client and mirrors the response back.
func proxyToBackend(c fiber.Ctx, target string) error {
	upstream := "http://" + target
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	c.Request().CopyTo(req)
	uri := c.Request().URI()
	req.SetRequestURI(upstream + string(uri.RequestURI()))
	req.Header.Del("Host")
	req.Header.SetHost(target)

	client := &fasthttp.HostClient{Addr: target}
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
