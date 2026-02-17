# ============================================================
# 🎭 PNJ Anonymous Bot — Optimized Multi-Stage Dockerfile
# ============================================================
# Stage 1: Build   → Compile Go binary with all optimizations
# Stage 2: Runtime → Minimal scratch-like image (~15MB)
# ============================================================

# ────────────────────────────────────────────────────────────
# Stage 1: Builder
# ────────────────────────────────────────────────────────────
FROM golang:1.23-alpine AS builder

# Install build dependencies (ca-certificates for SMTP TLS)
RUN apk add --no-cache ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy dependency files first (better layer caching)
COPY go.mod go.sum ./

# Download dependencies (cached unless go.mod/go.sum change)
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build with maximum optimizations:
#   -ldflags="-s -w"   → Strip debug info & symbols (~30% smaller)
#   -trimpath           → Remove file system paths from binary
#   CGO_ENABLED=0       → Static binary (no libc dependency)
#   GOARCH=amd64        → Target architecture
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=1.0.0" \
    -trimpath \
    -o /build/pnj-bot \
    ./cmd/bot/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -trimpath \
    -o /build/pnj-csbot \
    ./cmd/csbot/

# ────────────────────────────────────────────────────────────
# Stage 2: Runtime (Distroless-like minimal image)
# ────────────────────────────────────────────────────────────
FROM alpine:3.19 AS runtime

# Labels for container metadata
LABEL maintainer="PNJ Anonymous Bot Team"
LABEL description="Telegram Anonymous Bot for Politeknik Negeri Jakarta Students"
LABEL version="1.0.0"
LABEL org.opencontainers.image.source="https://github.com/pnj-anonymous-bot"
LABEL org.opencontainers.image.title="PNJ Anonymous Bot"
LABEL org.opencontainers.image.description="Anonymous chat & confession platform for PNJ students"

# Install runtime dependencies only
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    && rm -rf /var/cache/apk/*

# Set timezone to Asia/Jakarta (WIB)
ENV TZ=Asia/Jakarta

# Create non-root user for security
RUN addgroup -g 1001 -S pnjbot && \
    adduser -u 1001 -S pnjbot -G pnjbot -h /app -s /sbin/nologin

# Set working directory
WORKDIR /app

# Create data directory for SQLite database
RUN mkdir -p /app/data && chown -R pnjbot:pnjbot /app/data

# Copy binary from builder
COPY --from=builder /build/pnj-bot /app/pnj-bot
COPY --from=builder /build/pnj-csbot /app/pnj-csbot

# Copy timezone data from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy CA certificates for SMTP TLS connections
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Set ownership
RUN chown pnjbot:pnjbot /app/pnj-bot /app/pnj-csbot && \
    chmod +x /app/pnj-bot /app/pnj-csbot

# Switch to non-root user
USER pnjbot

# Expose health check port
EXPOSE 8080

# Health check — verify bot process is running
HEALTHCHECK --interval=30s --timeout=10s --start-period=15s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Volume for persistent database storage
VOLUME ["/app/data"]

# Set default environment variables
ENV DB_PATH=/app/data/pnj_anonymous.db
ENV BOT_DEBUG=false
ENV OTP_LENGTH=6
ENV OTP_EXPIRY_MINUTES=10
ENV MAX_SEARCH_PER_MINUTE=5
ENV MAX_CONFESSIONS_PER_HOUR=3
ENV MAX_REPORTS_PER_DAY=5
ENV AUTO_BAN_REPORT_COUNT=3

# Run the bot
ENTRYPOINT ["/app/pnj-bot"]
