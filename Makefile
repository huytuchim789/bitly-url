.PHONY: install dev dev-client build build-server build-client swag sqlc gen db-up db-migrate db-down db-reset dc-up dc-down dc-logs lint lint-server lint-client clean setup-hooks act-lint act-build act-security help

# === Install Everything ===

install: install-backend install-frontend install-tools install-hooks install-git-secrets gen
	@echo "========================================"
	@echo "  All set! Run 'make dev' to start."
	@echo "========================================"

install-backend:
	@echo "[backend] Installing Go dependencies..."
	cd server && go mod tidy

install-frontend:
	@echo "[frontend] Installing Node dependencies..."
	cd client && pnpm install

install-tools:
	@echo "[tools] Installing CLI tools..."
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/air-verse/air@latest

install-hooks:
	@echo "[hooks] Setting up git hooks..."
	git config core.hooksPath .husky

install-git-secrets:
	@echo "[security] Installing git-secrets..."
ifeq ($(OS),Windows_NT)
	@if not exist "$(USERPROFILE)\.git-secrets" ( \
		echo Downloading git-secrets... && \
		git clone https://github.com/awslabs/git-secrets.git "$(TEMP)\git-secrets" && \
		powershell -Command "& '$(TEMP)\git-secrets\install.ps1'" \
	)
else
	@if ! command -v git-secrets >/dev/null 2>&1; then \
		echo Installing git-secrets...; \
		git clone https://github.com/awslabs/git-secrets.git /tmp/git-secrets && \
		cd /tmp/git-secrets && \
		sudo make install; \
	fi
endif
	@echo "[security] Registering secret patterns..."
	git secrets --register-aws 2>/dev/null || true
	git secrets --add 'sk-[A-Za-z0-9]{32,}' 2>/dev/null || true
	git secrets --add 'ghp_[A-Za-z0-9_]{36}' 2>/dev/null || true
	git secrets --add 'gho_[A-Za-z0-9_]{36}' 2>/dev/null || true

# === Development ===

dev: db-up
	@echo "Starting server in dev mode..."
	cd server && go run ./cmd/main.go

dev-client:
	cd client && pnpm run dev

# === Build ===

build-server:
	cd server && go build -o bin/server ./cmd/main.go

build-client:
	cd client && pnpm run build

build: build-server build-client

# === Code Generation ===

swag:
	cd server && swag init -g cmd/main.go --output docs

sqlc:
	cd server && sqlc generate

gen: swag sqlc

# === Database ===

db-up:
	docker run -d --rm --name bitly-db \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_PASSWORD=postgres \
		-e POSTGRES_DB=bitly \
		-p 5432:5432 \
		postgres:16-alpine 2>/dev/null || echo "Container already running"

db-migrate:
	docker exec -i bitly-db psql -U postgres -d bitly < server/db/migrations/001_create_urls.sql

db-down:
	docker stop bitly-db 2>/dev/null || echo "Container not running"

db-reset: db-down db-up db-migrate

# === Docker Compose ===

dc-up:
	docker compose up --build -d

dc-down:
	docker compose down

dc-logs:
	docker compose logs -f

# === Lint & Clean ===

lint-server:
	cd server && go vet ./...

lint-client:
	cd client && pnpm run lint

lint: lint-server lint-client

clean:
	rm -rf server/bin
	rm -rf server/internal/dbgen
	rm -rf server/docs
	rm -rf client/.next
	rm -rf client/out

# === Help ===

help:
	@echo "Usage:"
	@echo "  make install       Install everything (deps + tools + hooks + git-secrets)"
	@echo "  make dev           Start server locally (auto-starts Postgres)"
	@echo "  make dev-client    Start Next.js dev server"
	@echo "  make build         Build both server and client"
	@echo "  make swag          Generate OpenAPI docs"
	@echo "  make sqlc          Generate Go code from SQL"
	@echo "  make gen           Run all code generators (swag + sqlc)"
	@echo "  make db-up         Start Postgres container"
	@echo "  make db-migrate    Run SQL migrations"
	@echo "  make dc-up         Start everything with Docker Compose"
	@echo "  make dc-down       Stop Docker Compose"
	@echo "  make lint          Run lint for both client and server"
