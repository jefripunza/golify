# syntax=docker/dockerfile:1.7
# ---------- Stage 1: build Vue 3 FE ----------
FROM node:22-alpine AS fe-builder

WORKDIR /fe

# install deps first for cache reuse
COPY web/package.json web/package-lock.json* ./
RUN npm ci --legacy-peer-deps || npm install --legacy-peer-deps

# bring the FE source
COPY web/ ./

# build production bundle → /fe/dist
RUN npm run build

# ---------- Stage 2: build Go backend (embeds web/dist via go:embed) ----------
FROM golang:1.25-alpine AS be-builder

WORKDIR /src

# Go module cache layer
COPY go.mod go.sum* ./
RUN go mod download

# full source (web/dist is copied from the FE stage so go:embed finds it)
COPY . .
COPY --from=fe-builder /fe/dist ./web/dist

# build a static, fully-linked single binary named "run"
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -trimpath -ldflags "-s -w" -o /out/run .

# ---------- Stage 3: minimal runner ----------
FROM alpine:3.20

# ca-certs for HTTPS ACME / outbound TLS, tzdata, and a non-root user
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S golify && adduser -S -G golify golify

WORKDIR /app
COPY --from=be-builder /out/run /app/run

# data/ holds the sqlite db and ssl/; mount this as a volume
RUN mkdir -p /app/data/ssl/letsencrypt /app/data/ssl/custom && \
    chown -R golify:golify /app

USER golify

# three ports:
#   80   → HTTP, ACME http-01 solver + API
#   443  → HTTPS, full SPA + API
#   3000 → plaintext dashboard (SPA only)
EXPOSE 80 443 3000

VOLUME ["/app/data"]

ENTRYPOINT ["/app/run"]
