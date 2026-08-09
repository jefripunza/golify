# TODO

## Status: Sprint D selesai — bcrypt auth + WebSocket xterm + podman control

## Task (selesai)

### D1 — bcrypt hardening
- [x] auth.go: hashPassword/checkPassword (bcrypt cost 10)
- [x] login handler verify bcrypt + lazy migration plaintext→bcrypt
- [x] POST /api/v1/auth/login (public)
- [x] PATCH /api/v1/auth/password (JWT, ganti password sendiri)
- [x] POST /api/v1/admin/auth/register (admin only)
- [x] Verified: wrong pass 401, change pass works, restore works

### D2 — WebSocket exec
- [x] ws.go: fasthttp native server di :20004 (GOTIFY_WS)
- [x] /ws/exec?token=<jwt>&service_id=<id> — auth via JWT query
- [x] podman exec -it golify-svc-<id> /bin/sh (fallback docker, lalu host /bin/sh)
- [x] /ws/health endpoint
- [x] Verified: handshake 101, 401 tanpa token

### D3 — FE xterm → WS
- [x] ServiceDetailView: WebSocket real ke :20004, onData → ws.send, onmessage → term.write
- [x] ws.close on unmount
- [x] vue-tsc + npm build OK

### D4 — podman start/stop/restart
- [x] setServiceStatus → containerAction (podman/docker CLI)
- [x] Fallback DB status flip bila runtime tidak ada (CI-safe)
- [x] POST .../services/:id/{start,stop,restart}
- [x] restart → DB status "running"
- [x] Verified: podman create golify-svc-1 real container

### D5 — Verify & push
- [x] go vet OK, vue-tsc OK
- [x] Smoke: login, start/stop/restart, ws health
- [ ] CI hijau

## Blocked
(none)

## Finish
- Auth bcrypt penuh (login, ganti password, register admin)
- Terminal xterm via WebSocket ke container real
- Kontrol container (start/stop/restart) via podman/docker CLI
- WS port terpisah :20004 (GOTIFY_WS env)