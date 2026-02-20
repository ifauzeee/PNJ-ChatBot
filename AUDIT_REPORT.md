# 🎭 PNJ Anonymous Bot — Laporan Audit Kode Komprehensif

**Tanggal Audit:** 20 Februari 2026  
**Auditor:** Senior Software Architect & Professional Code Auditor  
**Project:** PNJ Anonymous Bot (Telegram Bot Platform)  
**Tech Stack:** Go 1.24, PostgreSQL, SQLite, Redis, React (Vite), Docker, Prometheus  
**Total Lines of Code:** ~10,751 baris Go + ~1,200 baris JSX/CSS  

---

## 1. Executive Summary

PNJ Anonymous Bot adalah platform chat anonim berbasis Telegram yang dirancang khusus untuk mahasiswa Politeknik Negeri Jakarta (PNJ). Project ini memiliki arsitektur yang **cukup solid** dengan desain multi-layered (cmd → bot/csbot → service → database), worker pool system, circuit breaker pattern, dan monitoring via Prometheus.

### Verdict Keseluruhan: ⭐⭐⭐⭐ (4/5) — Good, dengan beberapa perbaikan penting yang diperlukan

| Aspek | Rating | Catatan |
|-------|--------|---------|
| Arsitektur | ⭐⭐⭐⭐ | Clean separation, multi-binary |
| Keamanan | ⭐⭐⭐ | **CRITICAL: Secrets exposed di `.env`** |
| Performa | ⭐⭐⭐⭐ | Worker pool, circuit breaker, connection pooling |
| Maintainability | ⭐⭐⭐⭐ | Modular code, good naming |
| Testing | ⭐⭐⭐ | Coverage kurang di handler layer |
| DevOps | ⭐⭐⭐⭐ | Docker multi-stage, helm-like YAML, Prometheus |
| Cleanup Needed | ⭐⭐⭐ | Binary artifacts, dead code, metrics tidak dipakai |

---

## 2. Kekuatan Project Saat Ini

### ✅ Arsitektur & Design Pattern
- **Clean Architecture**: Pemisahan layer yang baik (`cmd/` → `internal/bot/` → `internal/service/` → `internal/database/`)
- **Multi-binary**: Main bot (`pnj-bot`) dan CS bot (`pnj-csbot`) dipisahkan sebagai binary terpisah
- **Worker Pool Pattern**: Update processing menggunakan goroutine pool + channel-based queue (`updateQ`)
- **Per-User Lock**: `sync.Map` + `sync.Mutex` per user untuk mencegah race condition
- **Circuit Breaker**: Implementasi custom untuk Redis resilience
- **Graceful Shutdown**: Context-based cancellation + timeout-based wait

### ✅ Infrastruktur & DevOps
- **Multi-stage Dockerfile**: Builder + Runtime stage untuk ukuran image minimal
- **Docker Compose**: Dev + Production overlay pattern
- **Prometheus Metrics**: ~25 custom metrics terdefinisi
- **Health Check**: HTTP endpoint `/health`, `/ready`, `/metrics`
- **Automated Backup**: Scheduler via Ofelia cron

### ✅ Fitur Keamanan
- **Email Verification**: OTP via Brevo API dengan domain whitelist (`pnj.ac.id`)
- **Rate Limiting**: Per-action rate limiting via Redis
- **OTP Brute-force Protection**: Lock after 5 failed attempts (15 minute cooldown)
- **Profanity Filter**: Regex-based word filtering
- **Image Moderation**: Sightengine API integration
- **Auto-ban System**: Automatic ban after N reports
- **Evidence Logging**: Chat evidence stored in Redis for reports

### ✅ Code Quality
- **Structured Logging**: Uber's Zap logger with levels
- **Error Tracking**: Sentry integration
- **Input Validation**: Centralized validation module
- **Squirrel Query Builder**: Type-safe SQL building with DB abstraction (Postgres/SQLite)

---

## 3. Analisis Detail per Modul/Layer

### 3.1 `cmd/bot/main.go` & `cmd/csbot/main.go`
| Aspek | Status | Detail |
|-------|--------|--------|
| Signal handling | ✅ | `signal.NotifyContext` untuk SIGINT/SIGTERM |
| Resource cleanup | ✅ | `defer` untuk DB close, logger sync, Sentry flush |
| Error handling | ✅ | `logger.Fatal` untuk critical failures |

**Catatan:** CS Bot (`cmd/csbot/main.go`) tidak menutup database connection setelah selesai — `db.Close()` tidak dipanggil.

### 3.2 `internal/config/config.go`
| Aspek | Status | Detail |
|-------|--------|--------|
| Validation | ✅ | Comprehensive validation with warnings |
| Defaults | ✅ | Sensible defaults untuk semua config |
| Type safety | ✅ | Strict typing for all fields |

**Catatan:** Config tidak memvalidasi `RedisURL` sama sekali — Redis URL diambil langsung dari env di `redis.go` tanpa melalui `Config` struct.

### 3.3 `internal/bot/bot.go` (Core Bot)
| Aspek | Status | Detail |
|-------|--------|--------|
| Architecture | ✅ | Clean handler registration pattern |
| Concurrency | ✅ | Worker pool + per-user locks |
| Health server | ✅ | Full health/ready/metrics endpoints |
| Shutdown | ✅ | Graceful with timeouts |

**Masalah Ditemukan:**
1. **Double DB Close**: `bot.go` line 451 memanggil `b.db.Close()` tetapi `main.go` line 47 juga sudah `defer db.Close()` — ini akan menyebabkan double close.
2. **High-cardinality metrics**: `user_id` sebagai label di Prometheus histograms (`updateProcessDurationSeconds`, `userLockWaitSeconds`) — ini akan menyebabkan **cardinality explosion** di production.

### 3.4 `internal/service/` (Business Logic Layer)
| File | LOC | Status | Detail |
|------|-----|--------|--------|
| `auth.go` | 189 | ✅ | OTP flow solid |
| `chat.go` | 280 | ⚠️ | Queue matching bisa sangat lambat |
| `redis.go` | 210 | ✅ | Circuit breaker integration |
| `profile.go` | 186 | ✅ | Clean implementation |
| `confession.go` | 60 | ✅ | Simple and correct |
| `evidence.go` | 101 | ⚠️ | LTrim -20 overwrites data |
| `moderation.go` | 119 | ✅ | Proper API integration |
| `room.go` | 116 | ✅ | Clean room management |
| `gamification.go` | 51 | ✅ | Simple reward system |
| `profanity.go` | 46 | ⚠️ | `\b` word boundary tidak cocok untuk bahasa Indonesia |
| `customer_service.go` | 58 | ✅ | Clean delegation |

### 3.5 `internal/database/` (Data Access Layer)
| Aspek | Status | Detail |
|-------|--------|--------|
| Multi-DB support | ✅ | PostgreSQL + SQLite abstraction |
| Migration | ✅ | `golang-migrate` with embedded files |
| Query builder | ✅ | Squirrel for dynamic queries |
| Connection pool | ✅ | Configured max connections & lifecycle |

### 3.6 `dashboard/` (React Frontend)
| Aspek | Status | Detail |
|-------|--------|--------|
| Functionality | ✅ | Real-time health monitoring |
| API integration | ✅ | Auto-refresh every 5 seconds |
| Error handling | ✅ | Toast-style error notifications |
| Build | ✅ | Vite + Dockerfile for production |

---

## 4. Temuan Konsistensi & Sinkronisasi

### 4.1 Naming Convention
| Masalah | Detail | Severity |
|---------|--------|----------|
| Redis URL config inconsistency | Config struct tidak punya field `RedisURL` tapi `redis.go` langsung baca `os.Getenv("REDIS_URL")` | ⚠️ Medium |
| Email validation duplicate | `email.IsValidPNJEmail()` dan `validation.IsValidPNJEmail()` — dua implementasi berbeda untuk fungsi yang sama | ⚠️ Medium |
| `HealthResponse` struct duplicate | Didefinisikan di `bot.go` dan `csbot/bot.go` dengan field berbeda | 🔵 Low |

### 4.2 Error Handling Consistency
| Masalah | Detail | Severity |
|---------|--------|----------|
| Mixed error messaging language | Sebagian error dalam bahasa Indonesia ("gagal memeriksa sesi"), sebagian English ("user not found") | 🔵 Low |
| Inconsistent error wrapping | Beberapa method wrap error dengan `%w`, beberapa tidak | 🔵 Low |
| Silent error swallowing | Beberapa tempat menggunakan `_ = err` tanpa logging (contoh: `csbot/bot.go` line 118, 142) | ⚠️ Medium |

### 4.3 Parse Mode Inconsistency
| Masalah | Detail | Severity |
|---------|--------|----------|
| Mixed Markdown/HTML | `sendMessage()` uses Markdown parse mode, `sendMessageHTML()` uses HTML. Beberapa handler menggunakan keduanya secara tidak konsisten dalam satu flow | 🔵 Low |

### 4.4 Config-to-Service Desync  
| Masalah | Detail | Severity |
|---------|--------|----------|
| `REDIS_URL` bypass | `RedisService` membaca env langsung, bukan dari `Config` | ⚠️ Medium |
| `APP_ENV` not in config | Logger membaca `APP_ENV` langsung dari env | 🔵 Low |
| `DASHBOARD_PORT` unused in Go | Hanya dipakai di `docker-compose.yml`, tidak pernah diakses di Go code | 🔵 Low |

---

## 5. Daftar Bug & Issues (dengan tingkat keparahan)

### 🔴 CRITICAL

| # | Bug | Lokasi | Dampak | Solusi |
|---|-----|--------|--------|--------|
| C1 | **Secrets exposed dalam `.env` yang COMMITTED ke repo** | `.env` (line 2-3, 14, 16, 48-49) | Bot token, SMTP password, Brevo API key, Sightengine credentials, dan IP server **tersimpan di file `.env`** yang terlihat ada di repo. Meskipun `.gitignore` mencantumkan `.env`, file ini ada di filesystem dan berpotensi sudah pernah di-commit. | **SEGERA** rotate semua credentials. Verify git history, gunakan `git filter-branch` atau BFG Repo Cleaner. |
| C2 | **Double Database Close** | `bot.go:451` + `cmd/bot/main.go:47` | `b.db.Close()` dipanggil di `Bot.Start()` DAN `defer db.Close()` di main — menyebabkan double-close panic/error | Hapus `b.db.Close()` dari `bot.go:451` — biarkan main.go yang handle |
| C3 | **Prometheus high-cardinality labels** | `bot.go:35-50` | `user_id` sebagai label di histograms/counters = memory explosion. Dengan 1000 user, ini menghasilkan 1000+ time series per metric | Gunakan bucket/anonymized label, bukan raw user_id |

### 🟡 HIGH

| # | Bug | Lokasi | Dampak | Solusi |
|---|-----|--------|--------|--------|
| H1 | **CS Bot dan Main Bot health server conflict** | `csbot/bot.go:290` + `bot.go:255` | Keduanya listen di `:8080`. Dalam Docker ini OK (container terpisah), tapi gagal jika dijalankan lokal bersamaan | Buat port configurable via env |
| H2 | **OTP brute-force state doesn't persist** | `service/auth.go:37` | `otpAttempts` disimpan di `sync.Map` (in-memory). Restart bot = semua lockout state hilang. Attacker bisa memanfaatkan restart untuk bypass | Pindahkan ke Redis dengan TTL |
| H3 | **Chat queue linear scan O(n)** | `service/chat.go:54-102` | `SearchPartner` melakukan `LRange(0, -1)` → scan seluruh queue. Untuk antrian 1000+ user, ini sangat lambat dan memblock Redis | Gunakan Redis sorted set atau redesign matching algorithm |
| H4 | **Evidence LTrim bug** | `service/evidence.go:56` | `LTrim(ctx, key, -20, -1)` hanya menyimpan 20 pesan terakhir, tapi ini dijalankan setiap kali ada pesan baru = evidence bisa hilang sebelum report | Naikkan limit atau gunakan capped list size yang lebih besar |
| H5 | **CSBot DB connection leak** | `cmd/csbot/main.go` | `db` dibuat tapi tidak ada `defer db.Close()` | Tambah `defer db.Close()` setelah `database.New()` |
| H6 | **Profanity filter `\b` word boundary** | `service/profanity.go:33` | `\b` (word boundary) dalam regex tidak bekerja optimal untuk bahasa Indonesia yang sering menggabungkan kata (contoh: "anjingnya", "tolong") | Implementasi substring matching atau gunakan library khusus bahasa Indonesia |

### 🟠 MEDIUM

| # | Bug | Lokasi | Dampak | Solusi |
|---|-----|--------|--------|--------|
| M1 | **Redis Close called twice** | `bot.go:448-453` | `b.redisSvc.Close()` dipanggil di bot shutdown, tapi Redis connection juga bisa digunakan oleh goroutine yang belum selesai | Pastikan semua goroutine selesai sebelum close Redis |
| M2 | **`handleVoteCallback` double answerCallback** | `callbacks.go:351,362` | Jika `VotePoll` error, `answerCallback` dipanggil di line 351. Jika success, dipanggil lagi di line 362. Tapi line 19 juga ada `defer b.answerCallback(callback.ID, "")` = triple answer attempt | Hapus `defer` answerCallback di `handleCallback` atau handle di setiap callback handler |
| M3 | **`processReward` di setiap chat message** | `chat_handlers.go:219` | Setiap pesan chat memicu `RewardActivity` → DB update points/exp. Untuk user yang mengirim 100 pesan/menit, ini 100 DB writes | Batch atau rate-limit reward processing |
| M4 | **Gamification streak di setiap update** | `bot.go:489-497` | `UpdateStreak` dan `RewardActivity` dipanggil untuk SETIAP Telegram update, bukan hanya sekali per hari | Tambah cache/flag di Redis untuk daily-only check |
| M5 | **Broadcast tanpa rate limiting** | `admin_handlers.go:117,159` | `time.Sleep(50ms)` antar message ≈ 20 msg/second. Telegram API limit adalah ~30 msg/second per bot, tapi burst bisa kena flood control | Implementasi proper rate limiter untuk broadcast |

### 🔵 LOW

| # | Bug | Lokasi | Dampak | Solusi |
|---|-----|--------|--------|--------|
| L1 | **`UserStats` struct tidak dipakai** | `models/models.go:220-225` | Dead code | Hapus atau gunakan |
| L2 | **`escapeMarkdown` incomplete** | `bot/utils.go:52-60` | Tidak escape `~`, `>`, `\|` — karakter yang punya arti di Markdown V2 | Lengkapi atau switch ke HTML-only |
| L3 | **`maskEmail` edge case** | `bot/utils.go:39-50` | Jika email tidak mengandung `@`, mengembalikan email asli | Tambah handling untuk format invalid |

---

## 6. Dead Code, Dummy Files & Junk Findings

### 6.1 Dead Code

| Lokasi File | Jenis | Deskripsi | Baris | Rekomendasi |
|-------------|-------|-----------|-------|-------------|
| `internal/service/redis.go:113` | Dead Function | `GetFromQueue()` — tidak dipanggil di manapun dalam project | 12 baris | ❌ Hapus |
| `internal/resilience/retry.go:25` | Dead Function | `BroadcastRetryConfig()` — didefinisikan tapi tidak pernah digunakan | 6 baris | ❌ Hapus |
| `internal/resilience/retry.go:33` | Unused in Production | `Retry()` — hanya digunakan di test, tidak di production code | 21 baris | ⚠️ Tetap (utility future) |
| `internal/resilience/retry.go:56` | Unused in Production | `RetryWithResult[T]()` — hanya digunakan di test | 22 baris | ⚠️ Tetap (utility future) |
| `internal/resilience/circuit_breaker.go:45` | Unused in Production | `DefaultCircuitBreakerConfig()` — hanya digunakan di test | 7 baris | ⚠️ Tetap |
| `internal/validation/validation.go:65` | Dead Function | `ValidateCallbackData()` — tidak dipanggil di manapun | 3 baris | ❌ Hapus |
| `internal/validation/validation.go:57` | Dead Function | `validation.IsValidPNJEmail()` — tidak digunakan (duplikat `email.IsValidPNJEmail()`) | 6 baris | ❌ Hapus |
| `internal/validation/validation.go:69` | Unused in Production | `ContainsOnlyPrintable()` — hanya digunakan di test | 7 baris | ⚠️ Tetap |
| `internal/metrics/metrics.go:124` | Dead Metric | `ActiveUsersGauge` — didefinisikan tapi tidak pernah di-Set/Inc | 4 baris | ❌ Hapus atau implementasikan |
| `internal/metrics/metrics.go:129` | Dead Metric | `QueueSizeGauge` — didefinisikan tapi tidak pernah di-Set/Inc | 4 baris | ❌ Hapus atau implementasikan |
| `internal/metrics/metrics.go:54` | Dead Metric | `BlocksTotal` — tidak pernah di-increment | 4 baris | ❌ Hapus atau implementasikan |
| `internal/metrics/metrics.go:59` | Dead Metric | `AutoBansTotal` — tidak pernah di-increment | 4 baris | ❌ Hapus atau implementasikan |
| `internal/metrics/metrics.go:84` | Dead Metric | `CircleJoinsTotal` — hanya di `circle_handlers.go:128`, tapi `CircleLeavesTotal` tidak pernah digunakan | 4 baris | ❌ Hapus CircleLeavesTotal |
| `internal/models/models.go:220-225` | Dead Struct | `UserStats` — didefinisikan tapi tidak pernah diinstantiasi | 6 baris | ❌ Hapus |

### 6.2 File Sampah / Junk

| Lokasi File | Jenis | Deskripsi | Ukuran | Rekomendasi |
|-------------|-------|-----------|--------|-------------|
| `bin/pnj-bot` | Build Artifact | Compiled binary (Linux) | 22.7 MB | ❌ HAPUS — seharusnya di-gitignore |
| `bin/pnj-bot.exe` | Build Artifact | Compiled binary (Windows) | 22.7 MB | ❌ HAPUS — seharusnya di-gitignore |
| `bin/pnj-csbot` | Build Artifact | Compiled binary (Linux) | 20.0 MB | ❌ HAPUS — seharusnya di-gitignore |
| `bin/pnj-csbot.exe` | Build Artifact | Compiled binary (Windows) | 20.0 MB | ❌ HAPUS — seharusnya di-gitignore |
| `dashboard/node_modules/` | Dependency Folder | Local npm dependencies | ~besar | ❌ Pastikan di-gitignore |
| `dashboard/package-lock.json` | Lock File | 104KB lock file — OK jika committed, tapi periksa | 102 KB | ✅ Tetap (praktik build wajar) |
| `.env` | **SECRETS FILE** | Berisi real credentials, tokens, dan passwords | 1.3 KB | **🔴 CRITICAL — jangan sampai tercommit** |

### 6.3 Keterangan Grafis Summary

```
Dead Code:       ~82 baris (0.76% dari total)
Dead Metrics:    5 Prometheus metrics tidak dipakai
Junk Files:      ~86 MB (compiled binaries yang seharusnya tidak ada)
Duplicated Logic: 2 tempat (IsValidPNJEmail, HealthResponse struct)
```

---

## 7. Rekomendasi Prioritas

| No | Kategori | Rekomendasi | Prioritas | Estimasi Effort |
|----|----------|-------------|-----------|-----------------|
| 1 | 🔴 Security | **Rotate SEMUA credentials** (bot token, SMTP password, Brevo API key, Sightengine key). Verify git history untuk leak. | CRITICAL | 2-4 jam |
| 2 | 🔴 Security | Audit `.git` history, gunakan `git filter-branch` atau BFG untuk menghapus secrets dari commit history | CRITICAL | 1-2 jam |
| 3 | 🔴 Bug | Fix double `db.Close()` di `bot.go` — hapus line 451-453 | CRITICAL | 15 menit |
| 4 | 🔴 Performance | Ganti `user_id` label di Prometheus metrics menjadi anonymized/bucketed | HIGH | 1 jam |
| 5 | 🟡 Bug | Fix CS Bot `db.Close()` leak di `cmd/csbot/main.go` | HIGH | 15 menit |
| 6 | 🟡 Performance | Refactor chat queue dari `LRange(0, -1)` menjadi Redis Sorted Set | HIGH | 4-6 jam |
| 7 | 🟡 Security | Pindahkan OTP attempt tracking dari `sync.Map` ke Redis | HIGH | 2 jam |
| 8 | 🟡 Config | Pindahkan `REDIS_URL` parsing ke `config.go` agar konsisten | MEDIUM | 30 menit |
| 9 | 🟠 Cleanup | Hapus compiled binaries dari `bin/` | MEDIUM | 5 menit |
| 10 | 🟠 Cleanup | Hapus dead code (functions, metrics, structs yang tidak dipakai) | MEDIUM | 1 jam |
| 11 | 🟠 Performance | Rate-limit gamification updates (streak & reward maks 1x/hari) | MEDIUM | 2 jam |
| 12 | 🟠 Performance | Fix `handleVoteCallback` double-answer bug | MEDIUM | 30 menit |
| 13 | 🔵 Refactor | Konsolidasi `IsValidPNJEmail` ke satu tempat | LOW | 30 menit |
| 14 | 🔵 Refactor | Unify error language (semua Indonesian atau semua English) | LOW | 2 jam |
| 15 | 🔵 Testing | Tambah integration test untuk handler layer | LOW | 8-16 jam |
| 16 | 🔵 Feature | Implementasi `ActiveUsersGauge` dan `QueueSizeGauge` metrics | LOW | 1 jam |

---

## 8. Cleanup & Housekeeping Plan

### Phase 1: Emergency Cleanup (Hari 1)

```bash
# 1. Hapus compiled binaries
rm -rf bin/

# 2. Verify .gitignore includes bin/ (sudah ada ✅)
cat .gitignore | grep "bin/"

# 3. Pastikan .env TIDAK pernah ter-commit
git log --all --diff-filter=A -- .env
# Jika ada results, jalankan:
# git filter-branch --force --index-filter 'git rm --cached --ignore-unmatch .env' --prune-empty --tag-name-filter cat -- --all

# 4. Rotate ALL credentials setelah cleanup git history
# - Buat BOT_TOKEN baru via @BotFather
# - Buat CS_BOT_TOKEN baru via @BotFather
# - Reset SMTP_PASSWORD di Gmail
# - Generate Brevo API key baru
# - Generate Sightengine credentials baru
```

### Phase 2: Code Cleanup (Hari 2-3)

```bash
# Daftar file yang perlu diedit untuk menghapus dead code:

# 1. internal/service/redis.go — hapus fungsi GetFromQueue()
# 2. internal/resilience/retry.go — hapus BroadcastRetryConfig()
# 3. internal/validation/validation.go — hapus ValidateCallbackData(), IsValidPNJEmail()
# 4. internal/metrics/metrics.go — hapus ActiveUsersGauge, QueueSizeGauge, BlocksTotal, AutoBansTotal, CircleLeavesTotal
# 5. internal/models/models.go — hapus struct UserStats
# 6. internal/bot/bot.go — hapus b.db.Close() dan b.redisSvc.Close() di line 448-453

# Verifikasi setelah cleanup:
go build ./...
go test ./...
go vet ./...
```

### Phase 3: Refactor (Minggu 2)
1. Pindahkan `REDIS_URL` ke `Config` struct
2. Konsolidasi `IsValidPNJEmail` ke satu tempat
3. Fix high-cardinality Prometheus labels
4. Implementasi Redis-based OTP attempt tracking
5. Refactor chat queue ke Redis Sorted Set

---

## 9. Roadmap Pengembangan 3 Bulan ke Depan

### Bulan 1: Stabilitas & Keamanan
| Minggu | Task | Priority |
|--------|------|----------|
| 1 | Rotate credentials, cleanup git history, fix critical bugs (C1-C3) | CRITICAL |
| 2 | Fix high-severity bugs (H1-H6), cleanup dead code | HIGH |
| 3 | Refactor chat queue ke Sorted Set, fix medium bugs | HIGH |
| 4 | Tambah integration tests untuk handler layer | MEDIUM |

### Bulan 2: Performa & Fitur
| Minggu | Task | Priority |
|--------|------|----------|
| 5 | Implementasi Redis-based caching untuk user profiles | HIGH |
| 6 | Rate-limit gamification updates, batch reward processing | MEDIUM |
| 7 | Dashboard enhancement: user stats, confession analytics | MEDIUM |
| 8 | Implementasi confession pagination, search by content | MEDIUM |

### Bulan 3: Skalabilitas & Polish
| Minggu | Task | Priority |
|--------|------|----------|
| 9 | API Gateway untuk dashboard (REST API terpisah dari health endpoint) | MEDIUM |
| 10 | Implementasi message queue (NATS/RabbitMQ) untuk decouple broadcast | MEDIUM |
| 11 | CI/CD pipeline: lint, test, build, deploy via GitHub Actions | HIGH |
| 12 | Load testing, performance tuning, documentation update | MEDIUM |

---

## 10. Saran Tambahan

### 10.1 Testing Strategy
- **Unit Test Coverage Target:** Minimal 60% untuk `service/` layer
- **Integration Test:** Gunakan `testcontainers-go` untuk PostgreSQL + Redis test
- **Current Gap:** Handler layer (`internal/bot/`) hampir tidak memiliki test — ini adalah area dengan risiko regresi tertinggi

### 10.2 Security Hardening (2026 Best Practices)
1. **Secret Management**: Gunakan HashiCorp Vault atau Docker Secrets
2. **Rate Limiting Enhancement**: Implementasi sliding window rate limiter (bukan fixed window)
3. **Input Sanitization**: Tambahkan HTML entity encoding untuk semua user input sebelum dikirim
4. **CORS**: Health endpoint memiliki `Access-Control-Allow-Origin: *` — batasi ke domain dashboard saja
5. **Audit Logging**: Log semua admin actions (ban, broadcast, admin_poll) ke persistent storage

### 10.3 Scalability Considerations
1. **Horizontal Scaling**: Saat ini bot menggunakan in-memory `sync.Map` untuk user locks — ini tidak bisa dishare antar instance. Pindahkan ke Redis distributed lock jika butuh multi-instance.
2. **Database**: PostgreSQL connection pool (100 max) sudah baik. Pertimbangkan read replicas jika user base >10K.
3. **Redis**: Single instance Redis sudah cukup untuk traffic saat ini. Redis Cluster jika butuh HA.

### 10.4 Code Quality Tools yang Direkomendasikan
```bash
# Tools wajib untuk CI/CD
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install golang.org/x/vuln/cmd/govulncheck@latest

# Run quality checks
golangci-lint run ./...
gosec ./...
govulncheck ./...
```

### 10.5 Dependency Updates
| Dependency | Current | Action |
|-----------|---------|--------|
| Go | 1.24.0 | ✅ Latest |
| PostgreSQL (Docker) | 15-alpine | ⚠️ Consider upgrade ke 17 |
| Redis (Docker) | 7-alpine | ✅ Good |
| `go-telegram-bot-api/v5` | v5.5.1 | ✅ Latest |
| `prometheus/client_golang` | v1.23.2 | ✅ Latest |

### 10.6 Architecture Improvement Ideas
1. **Event-Driven Architecture**: Gunakan Redis Pub/Sub untuk real-time notifications antar mikroservice
2. **API Layer**: Pisahkan dashboard API endpoint ke service terpisah dari bot (saat ini health endpoint melayani keduanya)
3. **Message Broker**: Untuk broadcast ke puluhan ribu user, pertimbangkan queue-based approach (Redis Streams atau NATS JetStream)
4. **Observability Stack**: Lengkapi Prometheus dengan Grafana dashboard + Loki untuk log aggregation

---

*Laporan ini dibuat berdasarkan analisis statis seluruh source code project pada 20 Februari 2026. Untuk pertanyaan atau klarifikasi, silakan hubungi auditor.*
