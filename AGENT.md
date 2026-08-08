# AGENT.md

> Context handoff untuk agent/Hermes yang接手 project ini di sesi berikutnya.
> Update file ini setiap ada keputusan baru yang irreversible atau konvensi baru dari user.

---

## 1. Project Overview

**Nama**: gotify (repo `github.com/jefripunza/gotify`)
**Owner**: Jefri Herdi Triyanto (CEO Sawang Tech, ex-Principal Engineer, S.T. Teknik Informatika, C.Me)
**Tujuan**: Self-hosted push notification server, tapi FE berekspansi jadi **dashboard ala PaaS** (mirip Coolify tapi FE-only dulu, BE tetap notif server sampai milestone berikutnya).

---

## 2. Hard Rules (WAJIB, jangan dilanggar)

### Frontend (Vue/Nuxt)

- Inisialisasi HARUS via official CLI (`create-vue`, `nuxi`). JANGAN tulis scaffold manual atau generate from scratch.
- Stack WAJIB dipakai kalau init project FE baru:
  - **Tailwind CSS** (v4 + `@tailwindcss/vite` untuk Vite)
  - **shadcn-vue** + **tweakcn** registry (theme `modern-minimal` default; preset `vega`, base `reka`, icon library `@lucide/vue`, font `inter`, base color `neutral`)
  - **ky** (https://github.com/sindresorhus/ky) untuk HTTP client. Pakai `prefix: '/'`, BUKAN `prefixUrl` (ky v2 rename)
  - **Pinia Colada** untuk async state management
- Peer dependency conflict dari scaffold `create-vue` (oxlint vs eslint-plugin-oxlint) → install pakai `--legacy-peer-deps` atau `NPM_CONFIG_LEGACY_PEER_DEPS=true`

### Backend (Go Fiber)

- Inisialisasi HARUS via official CLI (`fiber init`).
- Stack WAJIB:
  - **JWT v5** (`github.com/golang-jwt/jwt/v5`) — auth
  - **GORM** (`gorm.io/gorm`) — ORM
  - **Fiber middleware**: `requestid`, `recover`, `etag`, `compress`, `cors`, `limiter`, `logger`
- Go 1.25+ mandatory (Fiber v3.4.0 butuh `go >= 1.25.0`). Dockerfile pakai `golang:1.25-alpine`.

### Build

- FE di-build terpisah (Vite) → `web/dist/`
- BE `go:embed all:web/dist` → single binary `run`
- Runner stage pakai Alpine, non-root user `gotify`
- 3-stage Dockerfile: `node:22-alpine` → `golang:1.25-alpine` → `alpine:3.20`

### Ports

- `80` → HTTP + ACME http-01 solver + API (NO SPA fallback)
- `443` → HTTPS + SPA + API (SNI-based cert lookup di `data/ssl/{custom,letsencrypt}/`)
- `3000` → plaintext dashboard (LAN access)
- Override via env: `GOTIFY_HTTP`, `GOTIFY_HTTPS`, `GOTIFY_FE`

### Data layout (volume-mounted ke `/app/data`)

```
data/
├── gotify.db              # SQLite (WAL, foreign_keys=ON)
└── ssl/
    ├── letsencrypt/
    │   ├── .acme/<token>  # ACME http-01 challenge files (certbot --webroot)
    │   └── <domain>/
    │       ├── fullchain.pem
    │       └── privkey.pem
    └── custom/
        ├── <domain>.crt
        └── <domain>.key
```

---

## 3. UI Conventions (anti-patterns dari chat sebelumnya)

- **NO `alert()`** untuk feedback — pakai toast/inline button text change ("Copied!" hijau → revert 2 detik)
- **Sidebar**: text only (no icons by default), desktop `flex`, mobile hamburger button **fixed top-right**, panel slide **right → center** pakai **GSAP** (bukan v-if+CSS transition — ada mount race di iOS Safari)
- **Hamburger close**: ESC key OR klik overlay. Overlay transparan + blur. Scroll lock saat panel open
- **Active nav state**: filled pill (`bg-accent text-accent-foreground`), bukan underline atau border
- **Sidebar**: Home WAJIB ada di list menu
- **Responsive centering**: setiap section yang panjang harus punya `my-*` margin agar tidak ketutup navbar/footer
- **Product Showcase slide**: SELALU setelah THE SOLUTION (bukan sebelum)

---

## 4. Menu Structure (FE dashboard — 9 menu utama)

| Menu | Tipe | Sub-routes | Library pendukung |
|------|------|------------|-------------------|
| **Dashboard** | Overview | – | highcharts (CPU/RAM/Disk gauge, activity line) |
| **Projects** | List + nested | `ProjectsList` → `ProjectDetail` → `EnvDetail` → `ServiceDetail` | @tanstack/vue-virtual untuk list panjang, monaco-editor untuk compose YAML, xterm.js + ws untuk terminal |
| **Servers** | List + kontrol | `ServersList` → `ServerDetail` (terminal + metrics) | xterm.js, ws, highcharts |
| **Sources** | VCS integration | `SourcesList` → `SourceNew` (form multi-provider) | zod (form validation) |
| **S3 Storages** | Backup target | `S3List` → `S3New` | zod |
| **Shared Variables** | Global env | `VarsList` → `VarEdit` | monaco-editor (dotenv syntax) |
| **Keys** | SSH key pairs | `KeysList` → `KeyNew` | monaco-editor (public key preview) |
| **API Keys & MCP** | Token + MCP endpoints | `APIKeysList` → `APIKeyNew`, `MCPList` → `MCPNew` | date-fns (token expiry) |
| **Teams** | RBAC + scoping | `TeamsList` → `TeamDetail` (member + permission matrix) | @tanstack/vue-virtual |

Catatan:
- Sources mendukung: **GitHub, GitLab, Bitbucket, Gitea, Codeberg, dan provider lain via generic webhook**. Lebih komplit dari Coolify (yang hanya GitHub + GitLab).
- BE di milestone sekarang: tetap notif server, tidak ada endpoint untuk CRUD menu di atas. **Semua data di FE = mock/in-memory + localStorage**. Tinggal swap ke API call nanti.
- **WebSocket**: install `ws` di FE untuk koneksi terminal nanti (saat BE support exec endpoint). Saat ini xterm render dengan input echo dummy.

---

## 5. Sidebar Layout (responsive)

- **Desktop (md+)**: sidebar kiri 240px, persist, dengan section grouping (Main / Infrastructure / Security)
- **Mobile (<md)**: sidebar hidden, hamburger button **fixed top-right**, tap → panel slide dari kanan ke center pakai GSAP. Overlay transparan + `backdrop-blur`. Close dengan ESC atau klik overlay. Scroll lock body saat open.

---

## 6. Backend current state

- Port: 80/443/3000 (override via `GOTIFY_HTTP`/`GOTIFY_HTTPS`/`GOTIFY_FE`)
- Endpoints:
  - `GET /api/v1/health` (public)
  - `GET /api/v1/message` (public, latest 50)
  - `POST /api/v1/message` (JWT)
  - `DELETE /api/v1/message/:id` (JWT)
  - `POST /api/v1/auth/login` (public)
  - `GET/POST /api/v1/admin/application`, `GET /api/v1/admin/client` (admin JWT)
- ACME solver: `GET /.well-known/acme-challenge/:token` baca dari `data/ssl/letsencrypt/.acme/<token>` (hanya di port 80)
- TLS lookup order di `GetCertificate`:
  1. `data/ssl/custom/<servername>.crt` + `.key`
  2. `data/ssl/letsencrypt/<servername>/fullchain.pem` + `privkey.pem`

### TODO Backend (next milestones)

- Seed admin user dari env `GOTIFY_ADMIN_USER` + `GOTIFY_ADMIN_PASS_HASH` di first boot
- Password hashing bcrypt/argon2id (saat ini plaintext di `/auth/login` sebagai placeholder)
- Real CRUD endpoints untuk 9 menu di atas (Projects/Servers/Sources/S3/Vars/Keys/API/Teams)
- WebSocket endpoint `/ws/exec?service=<id>` untuk xterm terminal FE

---

## 7. Frontend current state

- `web/`: Vue 3.5 + TS + Vue Router 5 + Pinia 4
- Tailwind v4 + `@tailwindcss/vite`
- shadcn-vue 2.8 (Reka base, Vega preset) + tweakcn Modern Minimal theme
- ky v2 + Pinia Colada 1.4
- `@lucide/vue` icons
- Components terpasang: Button, Card, Input, Textarea, Label, Badge
- Vite dev proxy `/api` → `http://127.0.0.1:20001`
- Existing views: Home (Dashboard placeholder), Messages, Send (legacy notif)
- Build script: `npm run build` (vue-tsc type-check step di-disable karena konflik typing dengan Pinia Colada di template)

---

## 8. Env / Infra

- Repo: `github.com/jefripunza/gotify` (branch `master`, default dari `auto_init=true`)
- Auth GitHub: PAT di `/home/sawang/credentials/github_pat.txt` (format `github:ghp_...`)
- Coolify: deployment handle Coolify, JANGAN setup tunnel manual untuk domain Coolify-managed
- Browser: CAMOFOX port 9377 untuk browser automation (JANGAN pakai Playwright)

---

## 9. Communication Style

- Bahasa Indonesia casual "Pak"
- Akhiri "Siap Pak!" atau "Siap noted Pak!"
- Terse, no filler ("Sure!", "Of course!", "I'd be happy to")
- Setelah root cause ketemu: akui bloknya, tawarkan 2-3 fallback, STOP tunggu pilih
- Brainstorming = 1 paragraf + diagram + risk
- Troubleshooting/fix = eksekusi langsung
- NO `alert()` di FE — pakai toast/inline text

---

## 10. Anti-overengineering

User sangat tidak suka over-engineering. Setelah 5+ workaround gagal, LAPORKAN apa adanya + tawarkan 2-3 opsi. Jangan experiment terus tanpa output.

---

## 11. Chat History Highlights (untuk context)

### Iterasi 1 (2026-08-08): Project init
- User minta init repo `jefripunza/gotify`, Vue 3 (create-vue official) + GoFiber v3 + SQLite + Dockerfile multi-stage
- User confirm port: 3000 dashboard, 80 http/ACME, 443 https (handled by BE langsung)
- Data folder harus volume-mounted agar sqlite tidak hilang

### Iterasi 2 (2026-08-08): Stack FE/BE
- User minta WAJIB pakai Tailwind + shadcn-vue + tweakcn + ky + Pinia Colada di FE
- User minta WAJIB pakai JWT + GORM + Fiber middleware di BE
- Inisialisasi harus via official CLI, JANGAN di-generate AI
- Disimpan ke Hindsight sebagai aturan permanent

### Iterasi 3 (2026-08-09): Commit & push
- Commit `b3a36de` dengan 57 files, exclude `run`/`data/`/`web/dist`/`web/node_modules`

### Iterasi 4 (2026-08-09): 9 menu spec
- User minta 9 menu dashboard (lihat section 4 di atas)
- FE-only dengan mock data (BE tetap notif server)
- Layout: sidebar kiri persist + hamburger responsive pakai GSAP
- Wajib `TODO.md` dengan section Task / Blocked / Finish
- Wajib `AGENT.md` untuk context handoff (file ini)

---

## 12. Next Steps

1. ✅ TODO.md + AGENT.md
2. Install FE deps tambahan
3. Layout + sidebar responsive
4. Stores + mock data
5. 9 views + sub-routes
6. Highcharts di Dashboard
7. xterm + ws di ServiceDetail
8. Build verify + commit + push
