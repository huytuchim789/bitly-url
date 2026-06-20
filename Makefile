.PHONY: install dev dev-client build build-server build-client build-local-server build-local-client swag db-up db-down db-reset dc-up-local dc-down-local dc-up-prod dc-down-prod lint lint-server lint-client clean help

# === Install Everything ===

install: install-backend install-frontend install-tools install-hooks install-git-secrets swag
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

dev: dc-up-local
	@echo "Starting server in dev mode..."
	cd server && go run ./cmd/main.go

dev-client:
	cd client && pnpm run dev

# === Docker Image Build ===

build-server:
	docker build -t bitly-url-server -f docker/server/Dockerfile server/

build-client:
	docker build -t bitly-url-client -f docker/client/Dockerfile client/

build: build-server build-client

# === Local Dev Build ===

build-local-server:
	cd server && go build -o bin/server ./cmd/main.go

build-local-client:
	cd client && pnpm run build

build-local: build-local-server build-local-client

# === Code Generation ===

swag:
	cd server && swag init -g cmd/main.go --output docs 2>/dev/null || true

# === Database ===

db-up:
	docker run -d --rm --name bitly-db \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_PASSWORD=postgres \
		-e POSTGRES_DB=bitly \
		-p 5432:5432 \
		postgres:16-alpine 2>/dev/null || echo "Container already running"

db-down:
	docker stop bitly-db 2>/dev/null || echo "Container not running"

db-reset: db-down db-up
	@echo "Run migration manually: docker exec -i bitly-db psql -U postgres -d bitly < server/db/migrations/001_init.sql"

# === Docker Compose ===

dc-up-local:
	docker compose -f docker/compose/compose.local.yaml up -d

dc-down-local:
	docker compose -f docker/compose/compose.local.yaml down

dc-up-prod:
	docker compose -f docker/compose/compose.prod.yaml up -d

dc-down-prod:
	docker compose -f docker/compose/compose.prod.yaml down

# === Lint & Clean ===

lint-server:
	cd server && go vet ./...

lint-client:
	cd client && pnpm run lint

lint: lint-server lint-client

clean:
	rm -rf server/bin
	rm -rf server/docs
	rm -rf client/.next
	rm -rf client/out

# === Help ===

help:
	@echo "Usage:"
	@echo "  make install        Install everything (deps + tools + hooks)"
	@echo "  make dev            Start server locally (auto-starts Postgres)"
	@echo "  make dev-client     Start Next.js dev server"
	@echo "  make build          Build Docker images for server + client"
	@echo "  make dc-up-local    Start infra only (db, redis, prometheus)"
	@echo "  make dc-up-prod     Start full production stack"
	@echo "  make swag           Generate OpenAPI docs"
	@echo "  make lint           Run linters"
