# TODO

## Status: Sprint C selesai — Login + BE CRUD 7 menu + FE full swap API

## Task (selesai)

### C1 — Model GORM 7 menu
- [x] models.go: Server, Source, S3Storage, SharedVariable, Key, ApiKey, McpEndpoint, Team + TeamMember
- [x] AutoMigrate semua model baru

### C2 — BE CRUD 7 menu (infra.go)
- [x] GET/POST/PATCH/DELETE untuk servers, sources, s3, variables, keys, api-keys, mcp, teams
- [x] helper generik: createGeneric, getGeneric, patchGeneric, deleteGeneric
- [x] registerInfra(v1) di api.go sebelum group auth

### C3 — Seed lengkap
- [x] seedAdmin: user admin/bismillah bila users kosong
- [x] seedServers 3, seedSources 3, seedS3 2, seedVariables 5, seedKeys 2, seedApiKeysAndMcp 2+2, seedTeams 2 (+members)

### C4 — FE Login + auth + guard
- [x] stores/auth.ts (login/logout via fetch + localStorage AUTH_KEY)
- [x] views/LoginView.vue (form admin/bismillah, error inline)
- [x] router guard beforeEnter → redirect /login (semua route kecuali /login)
- [x] Topbar: logout button + username + fix "Golify Dashboard"

### C5 — FE full swap mock → API
- [x] Semua 9 store pakai Colada useQuery + authed() ky + mapper DTO
- [x] Fallback mock di catch bila BE unreachable

### C6 — Verify & push
- [x] go vet OK, vue-tsc OK, npm build OK
- [x] Image rebuild + container smoke test:
  - health {app:golify,status:ok}
  - login admin/bismillah → token
  - 9 endpoint list 200 (projects 3, servers 3, sources 3, s3 2, variables 5, keys 2, api-keys 2, mcp 2, teams 2)
  - POST servers → 201
  - SPA fallback /login+/projects → index.html
- [x] Cleanup container + volume test

## Blocked
(none)

## Finish
- Feat besar: dashboard lengkap backend-backed (semua 9 menu)
- Login screen dengan user admin seeded otomatis
- /api/v1/{servers,sources,s3,variables,keys,api-keys,mcp,teams} CRUD JWT-protected
- Sistem seed otomatis hanya bila DB kosong (tidak overwrite data)