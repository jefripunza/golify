# TODO

Tracking task, blocker, dan finish untuk iterasi Gotify Dashboard FE (9 menu).

## Task

- [x] Tulis TODO.md dan AGENT.md di root repo
- [x] Install FE deps tambahan: xterm.js, @xterm/addon-fit, gsap, highcharts + highcharts-vue, ws, date-fns, monaco-editor, zod, @tanstack/vue-virtual
- [x] Layout: sidebar kiri persist + topbar tipis + hamburger responsive pakai GSAP (slide dari kanan ke tengah)
- [x] Pinia stores + mock data untuk 9 domain (projects, servers, sources, s3, vars, keys, api-mcp, teams)
- [x] 9 views utama: Dashboard, Projects, Servers, Sources, S3, SharedVars, Keys, APIKeysMCP, Teams
- [x] Sub-routes Projects: ProjectsList → ProjectDetail → EnvDetail → ServiceDetail (terminal xterm + ws placeholder)
- [x] Update router + sidebar menu dengan active state (filled pill pattern)
- [x] Highcharts di Dashboard untuk resource overview (CPU/RAM/Disk gauge + activity line)
- [x] Build verify (`npm run build`) — exit 0
- [x] Commit + push ke `jefripunza/gotify` branch master (commit `7ff2664`)

## Blocked

(none)

## Finish

### Build
- FE build: 0 error, bundle sizes:
  - index: 132 KB (50 KB gzip)
  - DashboardView: 368 KB (134 KB gzip, includes Highcharts)
  - ServiceDetailView: 354 KB (90 KB gzip, includes xterm.js)
- BE binary: 26 MB single binary dengan FE embedded (`go:embed`)

### Server smoke test
- `GET /api/v1/health` → 200 `{"app":"gotify","status":"ok"}`
- `GET /` di port 20002 → 200 SPA
- `GET /projects` (SPA fallback) → 200 SPA shell
- `GET /assets/index-*.js` → 200 `text/javascript`

### Routes (14 total)
- `/` (Dashboard)
- `/projects` → `/projects/:projectId` → `/projects/:projectId/:envId` → `/projects/:projectId/:envId/:serviceId`
- `/servers` → `/servers/:serverId`
- `/sources`
- `/s3`
- `/variables`
- `/keys`
- `/api-mcp`
- `/teams` → `/teams/:teamId`

### Stores (8 Pinia + 1 Colada)
- `useProjectsStore`, `useServersStore`, `useSourcesStore`, `useS3Store`, `useVarsStore`, `useKeysStore`, `useApiMcpStore`, `useTeamsStore` — semua persist via localStorage
- `useHealth` (Colada) untuk `/api/v1/health`

### Mock data
- 3 projects (sawang.tech-website, gotify, hindsight-agent-memory) masing-masing dengan envs + services
- 3 servers (laptop-jefri, vps-jakarta-1, legacy-build)
- 3 sources (GitHub, GitLab, Gitea)
- 2 S3 storages
- 5 shared variables (3 global, 1 project-scoped, 1 secret)
- 2 SSH keys
- 2 API keys + 2 MCP endpoints
- 2 teams (developers full access, viewers read-only)

### UI components
- shadcn primitives ditambah: `tabs`, `separator`, `dialog`, `select`
- Custom: `ClientOnly` wrapper, `Sidebar` (dengan GSAP animation), `Topbar` (hamburger fixed top-right), `useSidebar` composable
- 13 view files + 2 layout components + 1 composable

### Docs
- `AGENT.md` (9.5 KB) — full context handoff: hard rules, UI conventions, menu schema, next steps, chat history highlights
- `TODO.md` (file ini) — task tracker
