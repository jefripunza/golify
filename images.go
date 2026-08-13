package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// ─── Docker Registry HTTP API v2 image existence check ────────────────────
// The FE calls POST /api/v1/images/check (debounced) while the user types a
// Docker image. The backend resolves the registry (Docker Hub vs others),
// fetches the manifest and reports whether the image really exists and can
// be pulled. No auth / no pull performed — just a manifest HEAD/GET.

var (
	// rough but sufficient: name[/name...][:tag] or name[/name...]@sha256:…
	imageRefRe = regexp.MustCompile(`^[a-z0-9._/-]+(:[a-zA-Z0-9._-]+|@sha256:[a-f0-9]{64})?$`)
)

type imageCheckRequest struct {
	Image string `json:"image"`
}

type imageCheckResponse struct {
	Image     string `json:"image"`
	Exists    bool   `json:"exists"`
	Registry  string `json:"registry,omitempty"`
	Reference string `json:"reference,omitempty"`
	Error     string `json:"error,omitempty"`
}

func parseImageRef(ref string) (registry, repository, tag string, err error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", "", "", fmt.Errorf("image is required")
	}
	if !imageRefRe.MatchString(ref) {
		return "", "", "", fmt.Errorf("invalid image reference")
	}
	// split registry from repository: registry is the first segment if it
	// contains '.' or ':' or equals 'localhost'
	parts := strings.Split(ref, "/")
	registry = ""
	if len(parts) > 1 {
		first := parts[0]
		if strings.ContainsAny(first, ".:") || first == "localhost" {
			registry = first
			parts = parts[1:]
		}
	}
	if registry == "" {
		registry = "registry-1.docker.io"
	}
	repository = strings.Join(parts, "/")
	if repository == "" {
		return "", "", "", fmt.Errorf("invalid image reference")
	}
	// docker hub: library/<name> for single-segment repos
	if registry == "registry-1.docker.io" && !strings.Contains(repository, "/") {
		repository = "library/" + repository
	}
	tag = "latest"
	if i := strings.LastIndex(repository, "@"); i >= 0 {
		// digest reference — keep as is (repository@sha256:…)
		tag = repository[i:]
		repository = repository[:i]
	} else if i := strings.LastIndex(repository, ":"); i >= 0 {
		tag = repository[i+1:]
		repository = repository[:i]
	}
	return registry, repository, tag, nil
}

// registryHost returns the public hostname for a registry label.
func registryHost(registry string) string {
	switch registry {
	case "registry-1.docker.io":
		return "registry-1.docker.io"
	case "ghcr.io", "gcr.io", "quay.io", "docker.io", "registry.gitlab.com", "public.ecr.aws", "mcr.microsoft.com", "k8s.gcr.io", "registry.k8s.io":
		return registry
	default:
		return registry
	}
}

// checkImageExists queries the Docker Registry HTTP API v2:
// GET /v2/<repository>/manifests/<tag> with Accept: application/vnd.docker.distribution.manifest.v2+json
// Docker Hub requires a token first (GET /v2/ → WWW-Authenticate → fetch token → retry).
func checkImageExists(rawRef string) imageCheckResponse {
	resp := imageCheckResponse{Image: strings.TrimSpace(rawRef)}
	registry, repository, tag, err := parseImageRef(resp.Image)
	if err != nil {
		resp.Error = err.Error()
		return resp
	}
	resp.Registry = registry
	resp.Reference = fmt.Sprintf("%s:%s", repository, strings.TrimPrefix(tag, "@"))

	host := registryHost(registry)
	manifestURL := fmt.Sprintf("https://%s/v2/%s/manifests/%s", host, repository, tag)
	client := &http.Client{Timeout: 8 * time.Second}

	get := func(authHeader string) (*http.Response, error) {
		req, err := http.NewRequest(http.MethodGet, manifestURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.oci.image.index.v1+json")
		if authHeader != "" {
			req.Header.Set("Authorization", authHeader)
		}
		return client.Do(req)
	}

	// 1) try without auth first (works for most non-Docker-Hub registries)
	r, err := get("")
	if err != nil {
		resp.Error = fmt.Sprintf("cannot reach registry: %v", err)
		return resp
	}
	defer r.Body.Close()

	if r.StatusCode == http.StatusUnauthorized {
		// 2) Docker Hub / token-auth registries: parse WWW-Authenticate,
		//    fetch a bearer token, retry with it
		authHeader := r.Header.Get("WWW-Authenticate")
		if token, terr := fetchRegistryToken(authHeader, host, repository); terr == nil && token != "" {
			r.Body.Close()
			r2, err2 := get("Bearer " + token)
			if err2 != nil {
				resp.Error = fmt.Sprintf("cannot reach registry: %v", err2)
				return resp
			}
			defer r2.Body.Close()
			r = r2
		}
	}

	if r.StatusCode == http.StatusOK {
		resp.Exists = true
		return resp
	}
	// 401/403 after a token retry: the registry hides existence behind auth
	// (Docker Hub returns 401 for private/unknown repos, ghcr.io same) — a
	// successful token fetch that still ends in 401/403 means the image is
	// not pullable by anonymous users ⇒ treat as not exists.
	if r.StatusCode == http.StatusUnauthorized || r.StatusCode == http.StatusForbidden {
		resp.Error = "image not found or not publicly pullable"
		return resp
	}
	if r.StatusCode == http.StatusNotFound {
		resp.Error = "image not found in registry"
		return resp
	}
	resp.Error = fmt.Sprintf("registry returned HTTP %d", r.StatusCode)
	return resp
}

// fetchRegistryToken implements the bearer-token flow from RFC 7235 /
// Docker Registry v2: parse WWW-Authenticate: Bearer realm=…,service=…,scope=…
func fetchRegistryToken(wwwAuth, host, repository string) (string, error) {
	if !strings.HasPrefix(wwwAuth, "Bearer ") {
		return "", fmt.Errorf("unsupported auth scheme: %s", wwwAuth)
	}
	params := parseAuthParams(wwwAuth[len("Bearer "):])
	realm := params["realm"]
	if realm == "" {
		return "", fmt.Errorf("missing realm in WWW-Authenticate")
	}
	req, err := http.NewRequest(http.MethodGet, realm, nil)
	if err != nil {
		return "", err
	}
	q := req.URL.Query()
	if svc := params["service"]; svc != "" {
		q.Set("service", svc)
	}
	if scope := params["scope"]; scope != "" {
		q.Set("scope", scope)
	} else {
		q.Set("scope", "repository:"+repository+":pull")
	}
	req.URL.RawQuery = q.Encode()

	client := &http.Client{Timeout: 8 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint HTTP %d", res.StatusCode)
	}
	var tok struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(res.Body).Decode(&tok); err != nil {
		return "", err
	}
	if tok.Token != "" {
		return tok.Token, nil
	}
	return tok.AccessToken, nil
}

// parseAuthParams parses `key="value",key2="value2"` from WWW-Authenticate.
func parseAuthParams(s string) map[string]string {
	out := map[string]string{}
	for {
		s = strings.TrimLeft(s, " ,")
		if s == "" {
			break
		}
		var key, val string
		for len(s) > 0 && s[0] != '=' {
			key += string(s[0])
			s = s[1:]
		}
		s = strings.TrimLeft(s, "=")
		if len(s) > 0 && s[0] == '"' {
			s = s[1:]
			for len(s) > 0 && s[0] != '"' {
				val += string(s[0])
				s = s[1:]
			}
			s = strings.TrimLeft(s, "\"")
		} else {
			for len(s) > 0 && s[0] != ',' {
				val += string(s[0])
				s = s[1:]
			}
		}
		if key != "" {
			out[key] = val
		}
	}
	return out
}
