package main

import (
	"strings"

	"github.com/gofiber/fiber/v3"
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

	// 2) must have a service attached: the env-scoped Domain whose
	//    environment has at least one Service.
	var envIDs []string
	if err := db.Model(&Domain{}).Where("host IN ?", candidates).Pluck("environment_id", &envIDs).Error; err != nil || len(envIDs) == 0 {
		return 0
	}
	// domains registered without an environment are not attached to a service yet
	valid := make([]string, 0, len(envIDs))
	for _, eid := range envIDs {
		if eid != "" {
			valid = append(valid, eid)
		}
	}
	if len(valid) == 0 {
		return 0
	}
	var svc int64
	if err := db.Model(&Service{}).Where("environment_id IN ?", valid).Count(&svc).Error; err != nil || svc == 0 {
		return 0
	}
	return 1
}
