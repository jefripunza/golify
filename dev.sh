#!/usr/bin/env bash
# ── Golify dev mode ─────────────────────────────────────────────────────────
# Runs the Go backend (reads .env via godotenv) AND the Vite dev server
# (reads web/.env) with hot reload for the frontend.
#
#   ./dev.sh          → backend + frontend dev (hot reload)
#   ./dev.sh --fe     → frontend dev only (backend must already run)
#   ./dev.sh --be     → backend only (no frontend dev server)
#
# Frontend: http://localhost:5173 (or VITE_DEV_PORT in web/.env)
# Backend:  ports from .env (default 3000/80/443/8080)
set -euo pipefail

cd "$(dirname "$0")"

case "${1:-}" in
  --fe)
    echo "→ Frontend dev only (Vite hot reload on :5173)"
    exec npm --prefix web run dev
    ;;
  --be)
    echo "→ Backend only (ports from .env)"
    exec ./run
    ;;
  *)
    echo "→ Building backend…"
    go build -o run .
    echo "→ Backend (./run) + Frontend dev (Vite hot reload)"
    # Trap Ctrl+C to kill both
    trap 'kill 0' EXIT INT TERM
    ./run &
    npm --prefix web run dev
    ;;
esac
