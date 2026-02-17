.PHONY: build run clean tidy docker docker-prod docker-stop docker-logs docker-status docker-clean docker-backup

# ── Local Development ─────────────────────────────────────

# Build the bot binary
build:
	go build -o bin/pnj-bot.exe ./cmd/bot/

# Run the bot locally
run:
	go run ./cmd/bot/

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf data/

# Tidy dependencies
tidy:
	go mod tidy

# Download dependencies
deps:
	go mod download

# Build and run locally
dev: tidy build
	./bin/pnj-bot.exe

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
