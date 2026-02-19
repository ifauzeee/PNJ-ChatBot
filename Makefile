.PHONY: build run clean tidy docker docker-prod docker-stop docker-logs docker-status docker-clean docker-backup lint test-coverage help security quality-gate

# Default target
all: help

# ── Help ──────────────────────────────────────────────────
help:
	@echo "🎭 PNJ Anonymous Bot - Development Tools"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Local Development:"
	@echo "  build           Build bot binaries"
	@echo "  run             Run bot locally"
	@echo "  tidy            Clean up go.mod"
	@echo "  lint            Run golangci-lint"
	@echo "  security        Run gosec security scan"
	@echo "  test            Run all unit tests"
	@echo "  test-coverage   Run tests and show coverage"
	@echo "  quality-gate    Run all quality checks (pre-push)"
	@echo ""
	@echo "Docker Operations:"
	@echo "  docker          Build & start (Dev)"
	@echo "  docker-prod     Build & start (Prod)"
	@echo "  docker-stop     Stop all containers"
	@echo "  docker-logs     View live logs"
	@echo "  docker-status   Check health & status"
	@echo "  docker-clean    Reset environment"
	@echo ""

# ── Local Development ─────────────────────────────────────

# Build the bot binary
build:
	@echo "🏗️  Building binaries..."
	@mkdir -p bin
	go build -o bin/pnj-bot.exe ./cmd/bot/
	go build -o bin/pnj-csbot.exe ./cmd/csbot/

# Run the bot locally
run:
	go run ./cmd/bot/

# Linting
lint:
	@echo "🔍 Linting code..."
	golangci-lint run ./...

# Security Scan
security:
	@echo "🛡️  Running security scan..."
	gosec ./...

# Testing
test:
	@echo "🧪 Running tests..."
	go test -v ./...

# Test Coverage
test-coverage:
	@echo "📊 Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning up..."
	rm -rf bin/
	rm -rf data/*.db
	rm -f coverage.out coverage.html

# Tidy dependencies
tidy:
	go mod tidy

# Quality gate (all checks)
quality-gate:
	./scripts/quality-gate.sh

# ── Docker ────────────────────────────────────────────────

# Build & start in development mode
docker:
	docker compose up --build -d
	@echo ""
	@echo "✅ Bot deployed! Health: http://localhost:8080/health"

# Build & start in production mode
docker-prod:
	docker compose -f docker-compose.yml -f docker-compose.prod.yml up --build -d
	@echo ""
	@echo "✅ Production deployed! Health: http://localhost:8080/health"

# Stop all containers
docker-stop:
	docker compose down

# View live logs
docker-logs:
	docker compose logs -f --tail=100 pnj-bot

# Check container status & health
docker-status:
	@docker ps -a --filter "name=pnj-anonymous-bot" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}\t{{.Size}}"
	@echo ""
	@curl -s http://localhost:8080/health 2>/dev/null || echo "⚠️  Health endpoint not reachable"

# Restart containers
docker-restart:
	docker compose restart

# Remove all containers, volumes, images
docker-clean:
	docker compose down -v --rmi all

# Backup database from container
docker-backup:
	@mkdir -p backups
	docker cp pnj-anonymous-bot:/app/data/pnj_anonymous.db backups/pnj_anonymous_$$(date +%Y%m%d_%H%M%S).db
	@echo "✅ Backup saved to backups/"

# Rebuild without cache
docker-rebuild:
	docker compose build --no-cache
	docker compose up -d
