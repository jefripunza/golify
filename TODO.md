# TODO

Tracking task, blocker, dan finish untuk iterasi Gotify Dashboard FE (9 menu).

## Task (in-progress / planned)

- [x] Tulis TODO.md dan AGENT.md di root repo
- [x] Install FE deps tambahan: xterm.js, @xterm/addon-fit, gsap, highcharts + highcharts-vue, ws, date-fns, monaco-editor, zod, @tanstack/vue-virtual
- [x] Layout: sidebar kiri persist + topbar tipis + hamburger responsive pakai GSAP (slide dari kanan ke tengah)
- [x] Pinia stores + mock data untuk 9 domain (projects, servers, sources, s3, vars, keys, api-mcp, teams)
- [x] 9 views utama: Dashboard, Projects, Servers, Sources, S3, SharedVars, Keys, APIKeysMCP, Teams
- [x] Sub-routes Projects: ProjectsList → ProjectDetail → EnvDetail → ServiceDetail (terminal xterm + ws placeholder)
- [x] Update router + sidebar menu dengan active state (filled pill pattern, sesuai anti-pattern dari chat sebelumnya)
- [x] Highcharts di Dashboard untuk resource overview (CPU/RAM/Disk gauge + activity line)
- [x] Build verify (`npm run build` di web/) — exit 0
- [x] Commit + push ke `jefripunza/gotify` branch master

## Blocked

(none)

## Finish

- FE build: `npm run build` lulus 0 error, bundle sizes:
  - index: 132 KB (50 KB gzip)
  - DashboardView: 368 KB (134 KB gzip, includes Highcharts)
  - ServiceDetailView: 354 KB (90 KB gzip, includes xterm.js)
- BE binary: 26 MB single binary dengan FE embedded (`go:embed`)
- Server smoke test (port 20001/20002/20003):
  - `GET /api/v1/health` → 200 `{"app":"gotify","status":"ok"}`
  - `GET /` di port 20002 → 200 SPA
  - `GET /projects` (SPA fallback) → 200 SPA shell
  - `GET /assets/index-*.js` → 200 `text/javascript`
- 9 menu + 5 sub-routes terdaftar:
  - `/`, `/projects`, `/projects/:projectId`, `/projects/:projectId/:envId`, `/projects/:projectId/:envId/:serviceId`
  - `/servers`, `/servers/:serverId`
  - `/sources`, `/s3`, `/variables`
  - `/keys`, `/api-mcp`, `/teams`, `/teams/:teamId`
- Stores: `useProjectsStore`, `useServersStore`, `useSourcesStore`, `useS3Store`, `useVarsStore`, `useKeysStore`, `useApiMcpStore`, `useTeamsStore` — semua persist via localStorage
- Mock data lengkap untuk semua domain (3 projects, 3 servers, 3 sources, 2 S3, 5 vars, 2 keys, 2 API keys, 2 MCP endpoints, 2 teams)
