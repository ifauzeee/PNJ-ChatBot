#!/bin/bash
# ============================================================
# 🎭 PNJ Anonymous Bot — Deploy Script
# ============================================================
# Usage:
#   ./scripts/deploy.sh          → Build & start (development)
#   ./scripts/deploy.sh prod     → Build & start (production)
#   ./scripts/deploy.sh stop     → Stop all containers
#   ./scripts/deploy.sh logs     → View live logs
#   ./scripts/deploy.sh status   → Check container status
#   ./scripts/deploy.sh restart  → Restart containers
#   ./scripts/deploy.sh clean    → Remove everything
#   ./scripts/deploy.sh backup   → Backup database
# ============================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
COMPOSE_FILE="$PROJECT_DIR/docker-compose.yml"
COMPOSE_PROD="$PROJECT_DIR/docker-compose.prod.yml"
CONTAINER_NAME="pnj-anonymous-bot"
BACKUP_DIR="$PROJECT_DIR/backups"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

print_banner() {
    echo -e "${CYAN}"
    echo "╔══════════════════════════════════════════════════╗"
    echo "║   🎭  PNJ Anonymous Bot — Deploy Script         ║"
    echo "╚══════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

check_env() {
    if [ ! -f "$PROJECT_DIR/.env" ]; then
        echo -e "${RED}❌ .env file not found!${NC}"
        echo -e "${YELLOW}   Copy .env.example to .env and configure it:${NC}"
        echo "   cp .env.example .env"
        exit 1
    fi

    # Check BOT_TOKEN is set
    if grep -q "your_telegram_bot_token_here" "$PROJECT_DIR/.env"; then
        echo -e "${RED}❌ BOT_TOKEN is not configured in .env!${NC}"
        exit 1
    fi

    echo -e "${GREEN}✅ Environment file validated${NC}"
}

check_docker() {
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}❌ Docker is not installed!${NC}"
        exit 1
    fi

    if ! docker info &> /dev/null; then
        echo -e "${RED}❌ Docker daemon is not running!${NC}"
        exit 1
    fi

    echo -e "${GREEN}✅ Docker is available${NC}"
}

deploy_dev() {
    echo -e "${CYAN}🚀 Deploying in DEVELOPMENT mode...${NC}"
    check_env
    check_docker

    cd "$PROJECT_DIR"
    docker compose -f "$COMPOSE_FILE" up --build -d

    echo ""
    echo -e "${GREEN}✅ Bot deployed successfully!${NC}"
    echo -e "   📊 Health check: http://localhost:8080/health"
    echo -e "   📋 Metrics:      http://localhost:8080/metrics"
    echo -e "   📝 Logs:         docker compose logs -f pnj-bot"
}

deploy_prod() {
    echo -e "${CYAN}🚀 Deploying in PRODUCTION mode...${NC}"
    check_env
    check_docker

    cd "$PROJECT_DIR"
    docker compose -f "$COMPOSE_FILE" -f "$COMPOSE_PROD" up --build -d

    echo ""
    echo -e "${GREEN}✅ Bot deployed in production!${NC}"
    echo -e "   📊 Health check: http://localhost:8080/health"
    echo -e "   📋 Metrics:      http://localhost:8080/metrics"
}

stop() {
    echo -e "${YELLOW}🛑 Stopping containers...${NC}"
    cd "$PROJECT_DIR"
    docker compose down
    echo -e "${GREEN}✅ Containers stopped${NC}"
}

show_logs() {
    cd "$PROJECT_DIR"
    docker compose logs -f --tail=100 pnj-bot pnj-cs-bot
}

show_status() {
    echo -e "${CYAN}📊 Container Status:${NC}"
    docker ps -a --filter "name=pnj-" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}\t{{.Size}}"

    echo ""
    echo -e "${CYAN}🏥 Main Bot Health (8080):${NC}"
    curl -s http://localhost:8080/health 2>/dev/null | python3 -m json.tool 2>/dev/null || echo "   ⚠️  Main Bot endpoint not reachable"

    echo ""
    echo -e "${CYAN}🏥 CS Bot Health (8081):${NC}"
    curl -s http://localhost:8081/health 2>/dev/null | python3 -m json.tool 2>/dev/null || echo "   ⚠️  CS Bot endpoint not reachable"

    echo ""
    echo -e "${CYAN}💾 Volume Info:${NC}"
    docker volume inspect pnj-anonymous-bot-data --format '   Size: {{.Mountpoint}}' 2>/dev/null || echo "   ⚠️  Volume not found"
}

restart() {
    echo -e "${YELLOW}🔄 Restarting...${NC}"
    cd "$PROJECT_DIR"
    docker compose restart
    echo -e "${GREEN}✅ Restarted${NC}"
}

clean() {
    echo -e "${RED}🗑️  Cleaning up everything...${NC}"
    read -p "Are you sure? This will delete all data! (y/N): " confirm
    if [[ "$confirm" =~ ^[Yy]$ ]]; then
        cd "$PROJECT_DIR"
        docker compose down -v --rmi all
        echo -e "${GREEN}✅ Everything cleaned up${NC}"
    else
        echo "Cancelled."
    fi
}

backup() {
    echo -e "${CYAN}💾 Backing up database using internal script...${NC}"
    mkdir -p "$BACKUP_DIR"

    if docker exec "$CONTAINER_NAME" /app/scripts/backup.sh; then
        echo -e "${CYAN}📥 Syncing backups to host...${NC}"
        docker cp "$CONTAINER_NAME:/app/backups/." "$BACKUP_DIR/"
        echo -e "${GREEN}✅ Backups successfully created and synced to $BACKUP_DIR${NC}"
    else
        echo -e "${RED}❌ Backup script failed - check if container is running and DB is accessible${NC}"
    fi
}

# ── Main ──────────────────────────────────────────────────

print_banner

case "${1:-dev}" in
    dev|development)
        deploy_dev
        ;;
    prod|production)
        deploy_prod
        ;;
    stop)
        stop
        ;;
    logs|log)
        show_logs
        ;;
    status|info)
        show_status
        ;;
    restart)
        restart
        ;;
    clean|remove)
        clean
        ;;
    backup)
        backup
        ;;
    *)
        echo "Usage: $0 {dev|prod|stop|logs|status|restart|clean|backup}"
        exit 1
        ;;
esac
