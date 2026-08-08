# TODO

Tracking rebranding `gotify` → `golify` + GitHub Action CI workflow.

## Task

- [x] Tulis TODO.md update untuk rebranding + CI
- [x] Audit semua occurrence nama lama di repo
- [x] Rename GitHub repo di GitHub via API
- [x] Rename local folder + update git remote
- [x] Update Go module path + semua string di source (BE + FE + docs)
- [x] Update Docker image tag di `docker-compose.yaml`
- [ ] Create `.github/workflows/ci.yml` (FE build matrix Node 22 + BE build+vet)
- [x] Update AGENT.md + README.md
- [ ] Buat Telegram topic baru "Golify.dev" via Bot API (BLOCKED — bot token belum ada di env; Pak perlu aktifkan dulu atau rename manual di Telegram group settings → topic Gotify.dev → edit name)
- [x] Build verify (FE + BE) post-rebrand — npm run build OK, go build OK
- [x] Commit & push — pushed to `jefripunza/golify` (commit `5a5ad6d`), CI run in progress at https://github.com/jefripunza/golify/actions/runs/31285042404

## Blocked

(none)

## Finish

- GitHub repo: `github.com/jefripunza/gotify` → `github.com/jefripunza/golify` (rename via API)
- Local folder: `~/gotify` → `~/golify`, git remote updated
- Go module: `github.com/jefripunza/gotify` → `github.com/jefripunza/golify`
- Strings replaced (38 occurrences across 12 files): AppName, db filename, JWT issuer, health response, Docker image/tag/container/volume, user/group, mock data, terminal banner, docs
- CI workflow: `.github/workflows/ci.yml` — backend (go vet + go build + smoke-test) + frontend (matrix Node 22/24, build, upload artifact)
- Build verify: `npm run build` exit 0, `go build` 26 MB single binary with FE embedded
- Commit: `5a5ad6d rebrand: gotify → golify (repo, module, strings) + add ci workflow`