# bitly-url

Production-ready URL shortener with Go backend, Next.js frontend, Redis caching, click analytics, and Prometheus observability.

## Architecture

```
┌─────────┐    ┌──────────────┐
│ Browser │───▶│  Nginx (:80) │
└─────────┘    └──────┬───────┘
                      │
              ┌───────┴────────┐
              ▼                ▼
      ┌──────────────┐  ┌──────────────┐
      │  Client (:3000)│  │  Server (:8080)│
      │  Next.js      │  │  Gin + pgx    │
      └──────────────┘  └───────┬───────┘
                                │
                    ┌───────────┴───────────┐
                    ▼                       ▼
            ┌──────────────┐       ┌──────────────┐
            │  Redis (:6379)│       │  Postgres    │
            │  Cache +     │       │  (:5432)     │
            │  Rate Limit  │       │  urls +      │
            │              │       │  clicks      │
            └──────────────┘       └──────┬───────┘
                                          │
                                    ┌─────▼─────┐
                                    │ Prometheus │
                                    │  (:9090)   │
                                    └───────────┘
```

- **Nginx** routes `/api/*` and `/:short` (6-char codes) → server, everything else → client
- **Redis** caches shortened URLs (TTL 1h) and stores rate-limiter counters
- **Postgres** stores URLs and click events; migrations auto-run on first startup
- **Click tracking** uses an async batch worker (5s ticker / 100 events) — never blocks redirects
- **Prometheus** scrapes `/metrics` from server for observability

## Tech Stack

| Layer | Technology |
|-------|-----------|
| **Backend** | Go 1.26, Gin, pgxpool (pgx v5), go-redis/v9, Prometheus client_golang, swaggo, caarlos0/env, slog (stdlib) |
| **Frontend** | Next.js 15, React 19, TanStack Query, shadcn/ui, Tailwind CSS, ESLint (flat config), Prettier |
| **Infra** | Docker Compose (Postgres 16 + Redis 7 + server + client + Nginx + Prometheus), Husky hooks, GitHub Actions (CI + Security) |

## Prerequisites

- Go 1.26+ (see `.go-version`)
- Node.js 26+ (see `.node-version`)
- pnpm
- Docker + Docker Compose
- Make

## Quick Start

### Local Development (hàng ngày)

```bash
# Terminal 1: Start infra (db, redis, prometheus) + backend
make dev

# Terminal 2: Start frontend
cd client && pnpm dev
```

Backend at `http://localhost:8080` — Frontend at `http://localhost:3000`

### Hoặc step-by-step

```bash
# 1. Cài dependencies (chỉ lần đầu)
make install

# 2. Start infra (db + redis + prometheus)
docker compose -f docker/compose/compose.local.yaml up -d

# 3. Migration (auto-run, manual nếu cần)
docker exec -i bitly-url-db-1 psql -U postgres -d bitly < server/db/migrations/001_init.sql

# 4. Backend
cd server && go run ./cmd/main.go

# 5. Frontend
cd client && pnpm dev
```

### Production (Docker full stack)

```bash
# 1. Build Docker images
make build

# 2. Start full stack (db + redis + server + client + nginx + prometheus)
docker compose -f docker/compose/compose.prod.yaml up -d
```

Access via `http://localhost` (Nginx single entry point).

## Service Access

| URL | Service | Môi trường |
|-----|---------|-----------|
| `http://localhost` | Nginx reverse proxy | prod |
| `http://localhost:3000` | Next.js client (direct) | local |
| `http://localhost:8080` | Go server (direct) | local |
| `http://localhost:8080/swagger/index.html` | Swagger API docs | local + prod |
| `http://localhost:8080/metrics` | Prometheus metrics endpoint | local + prod |
| `http://localhost:9090` | Prometheus UI | local + prod |
| `http://localhost:5432` | Postgres | internal |
| `http://localhost:6379` | Redis | internal |

## Makefile Commands

| Lệnh | Mô tả |
|------|-------|
| `make install` | Install deps + tools + hooks + generate swagger |
| `make dev` | Start infra (db + redis + prometheus) + backend (`go run`) |
| `make dev-client` | Start frontend (`pnpm dev`) |
| `make dc-up-local` | Start infra only (db, redis, prometheus) |
| `make dc-down-local` | Stop infra |
| `make build` | Build Docker images (server + client) |
| `make dc-up-prod` | Start full production stack (db → redis → server → client → nginx → prometheus) |
| `make dc-down-prod` | Stop production stack |
| `make build-local-server` | Build Go binary locally |
| `make build-local-client` | Build Next.js locally |
| `make swag` | Regenerate Swagger docs |
| `make lint` | Run linters (`go vet` + `eslint`) |
| `make clean` | Clean build artifacts |
| `make help` | Show all commands |

## Docker Compose Files

| File | Mục đích |
|------|----------|
| `docker/compose/compose.local.yaml` | Local dev — chỉ db, redis, prometheus. Server/client chạy trực tiếp bằng `go run` / `pnpm dev` |
| `docker/compose/compose.prod.yaml` | Production — full stack gồm server + client + nginx. Cần build image trước bằng `make build` |

## Environment Variables

### Server

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server listen port |
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/bitly?sslmode=disable` | Postgres connection string |
| `REDIS_URL` | `redis://localhost:6379/0` | Redis connection string |
| `ENVIRONMENT` | `development` | Runtime environment (development/production) |
| `LOG_LEVEL` | `info` | Log level (debug/info/warn/error) |

### Client

| Variable | Default | Description |
|----------|---------|-------------|
| `NEXT_PUBLIC_API_URL` | (empty) | API base URL. Empty = relative requests (works through Nginx). Set to `http://localhost:8080` for direct access. |

## API Endpoints

| Method | Path | Rate Limit | Description |
|--------|------|------------|-------------|
| `POST` | `/api/shorten` | 10/min | Create a short URL |
| `GET` | `/api/urls` | 100/min | List all URLs (limit/offset pagination) |
| `GET` | `/api/urls/:short` | 100/min | Get original URL by short code |
| `GET` | `/:short` | 100/min | Redirect to original URL (302 Found) |
| `GET` | `/healthz` | — | Liveness check |
| `GET` | `/readyz` | — | Readiness check (DB + Redis ping) |
| `GET` | `/metrics` | — | Prometheus metrics |
| `GET` | `/debug/pprof/*` | — | Go pprof profiling |

### Shorten a URL

```bash
curl -X POST http://localhost:8080/api/shorten \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/very/long/url"}'
```

Response:

```json
{
  "id": "3e7b1c2a-8f5d-4b9a-8c1d-2e3f4a5b6c7d",
  "short": "aB3xY9",
  "original": "https://example.com/very/long/url",
  "clicks": 0,
  "created_at": "2026-06-20T12:00:00Z",
  "updated_at": "2026-06-20T12:00:00Z"
}
```

## Project Structure

```
├── server/                     # Go source code
│   ├── cmd/main.go
│   ├── internal/
│   │   ├── config/             # Env config (caarlos0/env)
│   │   ├── entity/             # URL, Click structs
│   │   ├── cache/              # Redis cache interface + impl
│   │   ├── repository/         # Interfaces + Postgres impl
│   │   ├── usecase/            # Business logic (+ tests)
│   │   ├── handler/            # Gin handlers (+ tests)
│   │   ├── middleware/         # RequestID, Logger, CORS, Error, Metrics, RateLimit
│   │   ├── metrics/            # Prometheus metric definitions
│   │   ├── pkg/errors/         # Typed AppError
│   │   ├── database/           # pgxpool setup
│   │   └── router/             # Routes + Swagger
│   ├── db/migrations/          # SQL migrations
│   └── go.mod
│
├── client/                     # Next.js source code
│   ├── src/
│   │   ├── pages/              # Next.js pages
│   │   ├── app/                # App providers, layout
│   │   ├── widgets/            # Composed UI widgets
│   │   ├── features/           # Feature slices
│   │   ├── entities/           # Domain entities + TanStack Query hooks
│   │   └── shared/             # UI kit (shadcn/ui), API client, types
│   └── package.json
│
├── docker/
│   ├── server/Dockerfile       # Server image (multi-stage Go build)
│   ├── client/Dockerfile       # Client image (multi-stage Next.js build)
│   ├── nginx/
│   │   ├── Dockerfile
│   │   └── default.conf        # Nginx config (reverse proxy)
│   ├── compose/
│   │   ├── compose.local.yaml  # Local dev: db + redis + prometheus
│   │   └── compose.prod.yaml   # Production: full stack
│   ├── prometheus/
│   │   └── prometheus.yml       # Prometheus scrape config
│   └── env/
│       ├── .env.local          # Local dev env template
│       └── .env.prod           # Production env template
│
├── .go-version               # Go version pin
├── .node-version             # Node.js version pin
├── .github/workflows/
│   ├── ci.yml                  # CI: lint + build + test
│   └── security.yml            # Security: gitleaks + trivy + checkov
├── .husky/                     # Git hooks
├── Makefile
└── README.md
```

## Docker Compose Services

### Local (`compose.local.yaml`)

| Service | Image | Ports | Health Check |
|---------|-------|-------|-------------|
| `db` | postgres:16-alpine | 5432 | pg_isready |
| `redis` | redis:7-alpine | 6379 | redis-cli ping |
| `prometheus` | prom/prometheus | 9090 | — |

### Production (`compose.prod.yaml`)

| Service | Image / Build | Ports | Depends On |
|---------|--------------|-------|------------|
| `db` | postgres:16-alpine | — | — |
| `redis` | redis:7-alpine | — | — |
| `server` | bitly-url-server (pre-built) | — | db, redis |
| `client` | bitly-url-client (pre-built) | — | server |
| `nginx` | Dockerfile (docker/nginx/) | 80 | server, client |
| `prometheus` | prom/prometheus | 9090 | — |

## Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| **302 redirect** | Prevents browser caching; allows updating target URL later |
| **Redis cache-first** | Cache hit → immediate redirect; cache miss → query DB + populate cache |
| **Async click tracking** | Batch worker (5s / 100 events) never blocks the redirect response |
| **Rate limiting** | Redis sliding window per-IP per-endpoint; separate limits for POST/GET |
| **Open redirect protection** | Blocks private IPs, loopback, and localhost destinations |
| **Short code** | 6-char alphanumeric (`crypto/rand`), collision detection with retry (10 attempts) |
| **UUID primary key** | Separate `id` (UUID) from `short` (user-facing code); clean FK for click tracking |
| **No authentication** | Public shortening — simplifies architecture |
