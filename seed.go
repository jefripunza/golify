package main

import (
	"gorm.io/gorm"
)

// seedIfEmpty inserts starter dashboard data when the DB has no projects yet.
// Only runs on a fresh volume so existing data is never overwritten.
func seedIfEmpty(db *gorm.DB) {
	var count int64
	if err := db.Model(&Project{}).Count(&count).Error; err != nil {
		return
	}
	if count > 0 {
		return
	}

	logPrint := func(s string) { println("[seed] " + s) }

	sawang := Project{Name: "sawang.tech-website", Description: "Company website & landing pages", SourceID: "gh_sawang_web"}
	sawangEnv := Environment{Name: "production", IsProduction: true}
	sawangEnv.Domains = []Domain{{Host: "sawang.tech"}, {Host: "www.sawang.tech"}}
	sawangEnv.Services = []Service{
		{Name: "landing", Kind: "container", Image: "ghcr.io/sawang-source-code/sawang-tech-landing:latest", Status: "running", CPU: 3.2, Memory: 96, Ports: []string{"3000"}},
	}
	sawang.Envs = []Environment{sawangEnv}

	gotifyProj := Project{Name: "golify", Description: "Self-hosted PaaS-style dashboard (this project)", SourceID: "src_github_org"}
	gotifyProjEnv := Environment{Name: "production", IsProduction: true}
	gotifyProjEnv.Domains = []Domain{{Host: "golify.sawang.tech"}}
	gotifyProjEnv.Services = []Service{
		{Name: "golify-server", Kind: "container", Image: "jefripunza/golify:latest", Status: "running", CPU: 1.4, Memory: 64, Ports: []string{"3000"}},
		{Name: "golify-db", Kind: "container", Image: "ghcr.io/vectorize-io/hindsight:latest-slim", Status: "running", CPU: 0.6, Memory: 128, Ports: []string{"9999"}},
	}
	gotifyProj.Envs = []Environment{gotifyProjEnv}

	hindsight := Project{Name: "hindsight-agent-memory", Description: "Second brain memory retrieval for agents", SourceID: "src_github_org"}
	hindsightEnv := Environment{Name: "staging", IsProduction: false}
	hindsightEnv.Services = []Service{
		{Name: "hindsight-api", Kind: "compose", ComposePath: "compose.yml", Status: "building", CPU: 8.4, Memory: 256, Ports: []string{"8686", "8676"}},
	}
	hindsight.Envs = []Environment{hindsightEnv}

	if err := db.Create(&sawang).Error; err != nil {
		logPrint("sawang seed failed: " + err.Error())
	}
	if err := db.Create(&gotifyProj).Error; err != nil {
		logPrint("golify seed failed: " + err.Error())
	}
	if err := db.Create(&hindsight).Error; err != nil {
		logPrint("hindsight seed failed: " + err.Error())
	}
	logPrint("seeded 3 projects")
}