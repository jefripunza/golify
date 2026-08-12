# golify

Self-hosted push notification server — Vue 3 SPA + Go (Fiber v3) backend with embedded frontend, JWT auth, GORM/SQLite storage.

## Stack

### Frontend (`web/`)

Initialized from official [`create-vue`](https://github.com/vuejs/create-vue) — **no AI-generated scaffolding**.

- **Vue 3** + TypeScript + Vue Router + Pinia
- **Tailwind CSS v4** (via `@tailwindcss/vite`)
- **shadcn-vue** (Reka base, Vega preset) + **tweakcn** registry (Modern Minimal theme)
- **ky** — HTTP client (`src/lib/api.ts`)
- **Pinia Colada** — async state (`useQuery`/`useMutation`/`useQueryCache` in `src/stores/messages.ts`)
- shadcn primitives in use: `Button`, `Card`, `Input`, `Textarea`, `Label`, `Badge`
- Icons via `@lucide/vue`

### Backend (`main.go`, `api.go`, `models.go`)

- **Go 1.25** + [Fiber v3](https://gofiber.io/) HTTP framework
- **GORM** (`gorm.io/gorm`) + `gorm.io/driver/sqlite` (mattn/go-sqlite3, CGO required)
- **JWT v5** (`github.com/golang-jwt/jwt/v5`) — HS256, 7-day expiry, random 32-byte signing key
- Fiber middleware stack: `requestid`, `recover`, `etag`, `compress`, `cors`, `limiter` (200 req/min), `logger`
- `go:embed all:web/dist` — frontend bundled into the Go binary at compile time

## Ports

| Port | Protocol | Purpose |
|------|----------|---------|
| 20001 | HTTP | Proxy HTTP + ACME solver (GOTIFY_HTTP) |
| 20002 | HTTPS | Proxy HTTPS — SPA + API, TLS in-process (GOTIFY_HTTPS) |
| 20003 | HTTP | Dashboard + API + WebSocket — unified single port, WS served directly by Go (GOTIFY_FE) |
| 8080  | HTTP+HTTPS | Proxy all-in-one (single port) for Cloudflare tunnel — simtaru.online / wajadi.online tests (GOTIFY_DUAL) |

All overridable via `.env` (see `.env.example`) or env vars: `GOTIFY_HTTP`, `GOTIFY_HTTPS`, `GOTIFY_FE`, `GOTIFY_DUAL`.

## Dev mode (hot reload)

```bash
./dev.sh          # backend (./run) + Vite dev server with HMR
./dev.sh --fe     # frontend only (hot reload on :5173)
./dev.sh --be     # backend only
```

- **Frontend** (`web/`): `npm run dev` → Vite on `:5173`, proxies `/api` → `GOTIFY_HTTP` and `/ws` → `GOTIFY_FE`. Edit any `.vue`/`.ts` → instant HMR, no reload needed.
- **Backend** (Go): edits require a rebuild + restart (`go build -o run . && ./run`). Config via `.env` (godotenv) or env vars.
- FE dev vars in `web/.env` (prefix `VITE_`): `VITE_DEV_PORT`, `VITE_DEV_PROXY_API`, `VITE_DEV_PROXY_WS` — see `web/.env.example`.
- BE vars in `.env` — see `.env.example`.


## Build pipeline (single binary)

```
[1] node:22-alpine       npm run build  →  web/dist/
[2] golang:1.25-alpine   go:embed web/dist + go build  →  run (single binary, FE bundled)
[3] alpine:3.20          copy run, run as non-root `golify`
```

The container image only carries the BE binary; no Node runtime in production.

## Layout

```
/app/
├── run              # single Go binary, FE embedded via go:embed
└── data/            # mounted as a Docker volume
    ├── golify.db          # SQLite (WAL mode, foreign_keys=ON)
    └── ssl/
        ├── letsencrypt/   # <domain>/fullchain.pem + privkey.pem
        │                  #   plus .acme/<token> for ACME http-01 solver
        └── custom/        # <domain>.crt + <domain>.key
```

## API

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET    | `/api/v1/health` | – | `{status,app}` |
| GET    | `/api/v1/message` | – | List latest 50 messages |
| POST   | `/api/v1/message` | JWT | Create message |
| DELETE | `/api/v1/message/:id` | JWT | Delete a message |
| POST   | `/api/v1/auth/login` | – | Exchange username/password for JWT |
| GET    | `/api/v1/admin/application` | Admin JWT | List apps |
| POST   | `/api/v1/admin/application` | Admin JWT | Create app (returns token) |
| GET    | `/api/v1/admin/client` | Admin JWT | List clients |

Send a message:

```bash
curl -X POST http://localhost/api/v1/message \
  -H 'Content-Type: application/json' \
  -d '{"title":"hi","message":"world","priority":3}'
```

## Run

```bash
docker compose up -d --build
```

- Dashboard: <http://localhost:3000>
- HTTPS: <https://localhost> (needs cert in `data/ssl/custom/<domain>.crt` + `.key` or `data/ssl/letsencrypt/<domain>/fullchain.pem` + `privkey.pem`)
- ACME: `certbot certonly --webroot -w /app/data/ssl/letsencrypt -d your.domain` (challenge files are served by the BE on port 80 at `/.well-known/acme-challenge/:token`)

## Local dev (without Docker)

```bash
# terminal 1 — FE (Vite proxies /api to the Go backend)
cd web && npm install --legacy-peer-deps && npm run dev

# terminal 2 — BE on non-privileged ports (no root needed)
go build -o run . && GOTIFY_HTTP=:20001 GOTIFY_HTTPS=:20003 GOTIFY_FE=:20002 ./run
```

## TODO

- Seed admin user from env (`GOTIFY_ADMIN_USER` + `GOTIFY_ADMIN_PASS_HASH`) on first boot
- Hash passwords with bcrypt/argon2id in `/api/v1/auth/login` (currently plaintext comparison as a placeholder)
