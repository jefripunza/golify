package main

import (
	"time"

	"gorm.io/gorm"
)

// seedIfEmpty inserts starter dashboard data the first time a fresh volume
// is mounted. Only runs when Projects is empty, so existing data is never
// overwritten. Also seeds a default admin user (username: admin / password: bismillah)
// when no users exist yet — for local first-boot convenience.
func seedIfEmpty(db *gorm.DB) {
	seedAdmin(db)
	seedProjects(db)
	seedServers(db)
	seedSources(db)
	seedS3(db)
	seedVariables(db)
	seedKeys(db)
	seedApiKeysAndMcp(db)
	seedTeams(db)
}

func logSeed(s string) { println("[seed] " + s) }

// ─── admin user ────────────────────────────────────────────────────────────
func seedAdmin(db *gorm.DB) {
	var count int64
	if err := db.Model(&User{}).Count(&count).Error; err != nil || count > 0 {
		return
	}
	// Passhash intentionally left as the plaintext value for the local
	// dev convenience user. Production should bcrypt.
	if err := db.Create(&User{Username: "admin", Passhash: "bismillah", Admin: true}).Error; err != nil {
		logSeed("admin user failed: " + err.Error())
		return
	}
	logSeed("admin user created (username: admin, password: bismillah)")
}

// ─── projects (sawang.tech-website, golify, hindsight-agent-memory) ────────
func seedProjects(db *gorm.DB) {
	var count int64
	if err := db.Model(&Project{}).Count(&count).Error; err != nil || count > 0 {
		return
	}

	sawang := Project{Name: "sawang.tech-website", Description: "Company website & landing pages", SourceID: "src_github_org"}
	sawangEnv := Environment{Name: "production", IsProduction: true}
	sawangEnv.Domains = []Domain{{Host: "sawang.tech"}, {Host: "www.sawang.tech"}}
	sawangEnv.Services = []Service{
		{Name: "landing", Kind: "container", Image: "ghcr.io/sawang-source-code/sawang-tech-landing:latest", Status: "running", CPU: 3.2, Memory: 96, Ports: []string{"3000"}},
	}
	sawang.Envs = []Environment{sawangEnv}

	golify := Project{Name: "golify", Description: "Self-hosted PaaS-style dashboard (this project)", SourceID: "src_github_org"}
	golifyEnv := Environment{Name: "production", IsProduction: true}
	golifyEnv.Domains = []Domain{{Host: "golify.sawang.tech"}}
	golifyEnv.Services = []Service{
		{Name: "golify-server", Kind: "container", Image: "jefripunza/golify:latest", Status: "running", CPU: 1.4, Memory: 64, Ports: []string{"3000"}},
		{Name: "golify-db", Kind: "container", Image: "ghcr.io/vectorize-io/hindsight:latest-slim", Status: "running", CPU: 0.6, Memory: 128, Ports: []string{"9999"}},
	}
	golify.Envs = []Environment{golifyEnv}

	hindsight := Project{Name: "hindsight-agent-memory", Description: "Second brain memory retrieval for agents", SourceID: "src_github_org"}
	hindsightEnv := Environment{Name: "staging", IsProduction: false}
	hindsightEnv.Services = []Service{
		{Name: "hindsight-api", Kind: "compose", ComposePath: "compose.yml", Status: "building", CPU: 8.4, Memory: 256, Ports: []string{"8686", "8676"}},
	}
	hindsight.Envs = []Environment{hindsightEnv}

	for _, p := range []Project{sawang, golify, hindsight} {
		if err := db.Create(&p).Error; err != nil {
			logSeed("project " + p.Name + " failed: " + err.Error())
		}
	}
	logSeed("seeded 3 projects")
}

// ─── servers ───────────────────────────────────────────────────────────────
func seedServers(db *gorm.DB) {
	var count int64
	if err := db.Model(&Server{}).Count(&count).Error; err != nil || count > 0 {
		return
	}
	rows := []Server{
		{Name: "hermes-pc", Host: "hermes-pc.lan", IP: "192.168.69.2", Region: "yogya", Provider: "self-hosted", Status: "online", CPU: 24.6, Memory: 6144, MemoryTotal: 16384, Disk: 47, Containers: 11},
		{Name: "gpu-node", Host: "gpu.lan", IP: "192.168.69.10", Region: "yogya", Provider: "self-hosted", Status: "online", CPU: 12.4, Memory: 4096, MemoryTotal: 24576, Disk: 31, Containers: 4},
		{Name: "cf-tunnel", Host: "tunnel-edge", IP: "10.0.0.5", Region: "global", Provider: "cloudflare", Status: "online", CPU: 0.4, Memory: 64, MemoryTotal: 512, Disk: 5, Containers: 0},
	}
	for i := range rows {
		if err := db.Create(&rows[i]).Error; err != nil {
			logSeed("server failed: " + err.Error())
		}
	}
	logSeed("seeded 3 servers")
}

// ─── sources ───────────────────────────────────────────────────────────────
func seedSources(db *gorm.DB) {
	var count int64
	if err := db.Model(&Source{}).Count(&count).Error; err != nil || count > 0 {
		return
	}
	rows := []Source{
		{Name: "Sawang Source Code (org)", Provider: "github", URL: "https://github.com/sawang-source-code", IsGlobal: true},
		{Name: "jefripunza (personal)", Provider: "github", URL: "https://github.com/jefripunza", IsGlobal: true},
		{Name: "Sawang Tech Infra", Provider: "github", URL: "https://github.com/sawang-source-code/sawang-infra", IsGlobal: false},
	}
	for i := range rows {
		if err := db.Create(&rows[i]).Error; err != nil {
			logSeed("source failed: " + err.Error())
		}
	}
	logSeed("seeded 3 sources")
}

// ─── s3 ────────────────────────────────────────────────────────────────────
func seedS3(db *gorm.DB) {
	var count int64
	if err := db.Model(&S3Storage{}).Count(&count).Error; err != nil || count > 0 {
		return
	}
	rows := []S3Storage{
		{Name: "primary-backup", Endpoint: "https://s3.amazonaws.com", Region: "ap-southeast-1", Bucket: "sawang-tech-primary", AccessKeyID: "AKIA***PRIMARY", IsDefault: true},
		{Name: "offsite-mirror", Endpoint: "https://nyc3.digitaloceanspaces.com", Region: "nyc3", Bucket: "sawang-offsite", AccessKeyID: "DO***OFFSITE", IsDefault: false},
	}
	for i := range rows {
		if err := db.Create(&rows[i]).Error; err != nil {
			logSeed("s3 failed: " + err.Error())
		}
	}
	logSeed("seeded 2 s3 storages")
}

// ─── variables ─────────────────────────────────────────────────────────────
func seedVariables(db *gorm.DB) {
	var count int64
	if err := db.Model(&SharedVariable{}).Count(&count).Error; err != nil || count > 0 {
		return
	}
	rows := []SharedVariable{
		{Key: "GOLIFY_DOMAIN", Value: "golify.sawang.tech", IsSecret: false, Scope: "global"},
		{Key: "JWT_SIGN_KEY", Value: "<runtime-crypto-rand-32-byte>", IsSecret: true, Scope: "global"},
		{Key: "DB_PATH", Value: "/app/data/golify.db", IsSecret: false, Scope: "global"},
		{Key: "CLOUDFLARE_TUNNEL_ID", Value: "9c46bad2-7ed3-4d8e-bd44-...", IsSecret: true, Scope: "global"},
		{Key: "POSTGRES_URL", Value: "postgres://localhost:5432/svc", IsSecret: true, Scope: "service", ScopeRef: 2},
	}
	for i := range rows {
		if err := db.Create(&rows[i]).Error; err != nil {
			logSeed("variable failed: " + err.Error())
		}
	}
	logSeed("seeded 5 variables")
}

// ─── ssh keys ──────────────────────────────────────────────────────────────
func seedKeys(db *gorm.DB) {
	var count int64
	if err := db.Model(&Key{}).Count(&count).Error; err != nil || count > 0 {
		return
	}
	rows := []Key{
		{Name: "hermes-deploy", PublicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI... deploy@hermes", Fingerprint: "SHA256:hermesABC..."},
		{Name: "ci-runner", PublicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI... runner@ci", Fingerprint: "SHA256:ciRunner..."},
	}
	for i := range rows {
		if err := db.Create(&rows[i]).Error; err != nil {
			logSeed("key failed: " + err.Error())
		}
	}
	logSeed("seeded 2 ssh keys")
}

// ─── api keys + mcp ────────────────────────────────────────────────────────
func seedApiKeysAndMcp(db *gorm.DB) {
	var count int64
	if err := db.Model(&ApiKey{}).Count(&count).Error; err != nil || count > 0 {
		return
	}
	now := time.Now()
	apiRows := []ApiKey{
		{Name: "github-actions-ci", Prefix: "ghk_ci_", Scopes: []string{"projects:write", "deploy"}, LastUsedAt: &now},
		{Name: "strix-pentest", Prefix: "ghk_strix_", Scopes: []string{"read"}},
	}
	for i := range apiRows {
		if err := db.Create(&apiRows[i]).Error; err != nil {
			logSeed("apikey failed: " + err.Error())
		}
	}
	mcpRows := []McpEndpoint{
		{Name: "hindsight", URL: "http://127.0.0.1:9999/mcp", Transport: "http", ApiKeyID: apiRows[0].ID, Enabled: true},
		{Name: "9router", URL: "https://ai.jefripunza.com/mcp", Transport: "sse", ApiKeyID: apiRows[1].ID, Enabled: true},
	}
	for i := range mcpRows {
		if err := db.Create(&mcpRows[i]).Error; err != nil {
			logSeed("mcp failed: " + err.Error())
		}
	}
	logSeed("seeded 2 api keys + 2 mcp endpoints")
}

// ─── teams ─────────────────────────────────────────────────────────────────
func seedTeams(db *gorm.DB) {
	var count int64
	if err := db.Model(&Team{}).Count(&count).Error; err != nil || count > 0 {
		return
	}
	rows := []Team{
		{Name: "Core", Description: "Owner-level access", Permissions: `{"projects":"*","servers":"*","sources":"*","s3":"*","variables":"*","keys":"*","api-mcp":"*","teams":"*"}`},
		{Name: "Developers", Description: "Dev team — project read/write", Permissions: `{"projects":"write","servers":"read","sources":"read","s3":"read","variables":"read","keys":"read","api-mcp":"read","teams":"read"}`},
	}
	for i := range rows {
		if err := db.Create(&rows[i]).Error; err != nil {
			logSeed("team failed: " + err.Error())
			continue
		}
		members := []TeamMember{
			{TeamID: rows[i].ID, Email: "jefri@sawang.tech", Role: "owner", JoinedAt: now()},
		}
		if rows[i].Name == "Developers" {
			members = append(members, TeamMember{TeamID: rows[i].ID, Email: "dev@sawang.tech", Role: "developer", JoinedAt: now()})
		}
		for j := range members {
			if err := db.Create(&members[j]).Error; err != nil {
				logSeed("team member failed: " + err.Error())
			}
		}
	}
	logSeed("seeded 2 teams")
}

func now() time.Time { return time.Now() }