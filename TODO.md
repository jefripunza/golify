# TODO

## Status: Local dev live — https://golify.jefripunza.com via Cloudflare Tunnel

## Task (selesai)

### E — Run lokal + tunnel
- [x] vite.config.ts: tambah proxy /ws → :20004 (ws:true), allowedHosts golify.jefripunza.com, host 0.0.0.0
- [x] package.json dev: vite --host 0.0.0.0
- [x] ServiceDetailView: WS pakai location.host (same-origin, lewat proxy)
- [x] Go build ./run
- [x] Run ./run :20001/:20003/:20002/:20004 (BE+WS)
- [x] Run npm run dev (Vite :5173)
- [x] Tunnel AI Agent ingress golify.jefripunza.com → http://127.0.0.1:5173 (PUT v41)
- [x] DNS CNAME golify.jefripunza.com → 9c46bad2.cfargotunnel.com
- [x] Restart cloudflared-tunnel

### Verifikasi
- https://golify.jefripunza.com/ → 200
- https://golify.jefripunza.com/api/v1/health → {"app":"golify","status":"ok"}
- https://golify.jefripunza.com/api/v1/auth/login admin/bismillah → token
- https://golify.jefripunza.com/ws/health → {"app":"golify-ws","status":"ok"}
- WS handshake :443 → HTTP/1.1 101 Switching Protocols

## Blocked
(none)

## Finish
- Local FE+BE live, publik lewat Cloudflare Tunnel
- Vite proxy: /api → BE :20001, /ws → BE :20004
- WS xterm bisa diakses dari browser publik