# TODO

## Status: Sprint B selesai — BE CRUD Projects + FE swap ke API

## Task (selesai)

### A: Cleanup & CI
- [x] Rebuild image `localhost/golify:test` (pure-Go sqlite driver)
- [x] Smoke test container — health OK, SPA OK, POST 401, message [] (non-root user + `:U` volume flag)
- [x] Hapus db lama `gotify.db` dari volume
- [x] CI workflow — go vet + go build + smoke test green (fix: stub web/dist before go vet)
- [x] CI status: `31287444218` completed success (commit `3b6bf37`)
- [x] fix `main.go`: CGO → `glebarez/sqlite` pure-Go; Dockerfile `/app/data` chmod 777; compose volume `:U`

### B — BE CRUD Projects
- [x] B1: Model GORM Project → Environment → Service → Domain (`models.go`) + AutoMigrate
- [x] B2: REST API `/api/v1/projects` (CRUD) + nested envs + services + start/stop (`projects.go`)
- [x] B3: Seed 3 project bila DB kosong (`seed.go`)
- [x] B4: FE projects store swap mock → Colada query + ky authed() + DTO mapper (`api.ts`, `stores/index.ts`)
- [x] B5: Build verify (go vet + vue-tsc clean), smoke test end-to-end, image rebuild, cleanup test container/volume

## Blocked

(none)

## Finish

- **BE**: `models.go` + `projects.go` + `seed.go` — hierarki Project→Env→Service→Domain lengkap (JWT-protected)
  - `GET/POST /api/v1/projects/`, `GET/DELETE /api/v1/projects/:id` (nested preload envs+services+domains)
  - `GET/POST /:projectId/environments`, `GET/POST /.../services`, `POST start/stop`
  - Fix penting Fiber v3: login route HARUS decl sebelum `auth := v1.Group("", requireAuth)` (empty-prefix group middleware menimpa route setelahnya)
- **FE**: projects store pakai `useQuery` Colada + fallback mock; `authed()` ky instance pakai Bearer token
- Image `localhost/golify:test` build + container smoke test lulus
- Container `golify-test` + volume test dihapus; volume `golify-data` (data real) dipertahankan