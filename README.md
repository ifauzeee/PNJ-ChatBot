# 🎭 PNJ Anonymous Bot

> Bot Telegram anonim khusus untuk mahasiswa **Politeknik Negeri Jakarta**

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://golang.org)
[![Telegram Bot API](https://img.shields.io/badge/Telegram-Bot%20API-26A5E4?style=flat-square&logo=telegram&logoColor=white)](https://core.telegram.org/bots/api)
[![SQLite](https://img.shields.io/badge/SQLite-Database-003B57?style=flat-square&logo=sqlite&logoColor=white)](https://sqlite.org)

## ✨ Fitur

### 🔐 Verifikasi Email
- Khusus domain `@mhsw.pnj.ac.id` dan `@stu.pnj.ac.id`
- OTP 6-digit via email dengan template HTML premium
- Anti-duplikat email

### 🔍 Anonymous Chat
- `/search` — Cari partner chat anonim
- `/search [jurusan]` — Filter berdasarkan jurusan
- `/next` — Skip ke partner baru
- `/stop` — Hentikan chat
- Support: teks, foto, stiker, voice, video, dokumen, GIF

### 💬 Confession Board
- `/confess` — Kirim confession anonim
- `/confessions` — Lihat 10 confession terbaru
- Reaction system (❤️ 😂 😢 😮 🔥)
- Rate limiting (3 confession/jam)

### 📢 Whisper
- Kirim pesan anonim ke seluruh mahasiswa di jurusan tertentu
- Menampilkan gender & jurusan pengirim (tanpa identitas)

### 👤 Profil & Statistik
- `/profile` — Lihat profil kamu
- `/stats` — Statistik interaksi
- `/edit` — Edit gender/jurusan
- Nama anonim otomatis (contoh: MysteriousFox42)

### 🛡️ Keamanan
- `/report` — Laporkan partner
- `/block` — Block partner
- Auto-ban setelah 3 report
- Rate limiting semua fitur

## 🏛️ Jurusan PNJ

| Emoji | Jurusan |
|-------|---------|
| 🏗️ | Teknik Sipil |
| ⚙️ | Teknik Mesin |
| ⚡ | Teknik Elektro |
| 💻 | Teknik Informatika & Komputer |
| 🎨 | Teknik Grafika & Penerbitan |
| 📊 | Akuntansi |
| 📈 | Administrasi Niaga |
| 🎓 | Pascasarjana |

## 🚀 Quick Start

### Prerequisites
- Go 1.24+
- Telegram Bot Token (dari [@BotFather](https://t.me/BotFather))
- Brevo API Key (untuk kirim OTP email)

### Setup

1. **Clone & masuk ke directory**
```bash
git clone https://github.com/ifauzeee/PNJ-ChatBot.git pnj-anonymous-bot
cd pnj-anonymous-bot
```

2. **Copy environment file**
```bash
cp .env.example .env
```

3. **Edit `.env`** dengan konfigurasi kamu:
```env
BOT_TOKEN=your_telegram_bot_token
LOG_LEVEL=info
MAX_UPDATE_WORKERS=16
MAX_UPDATE_QUEUE=256
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your_email@gmail.com
SMTP_PASSWORD=your_app_password
SMTP_FROM=PNJ Anonymous Bot <your_email@gmail.com>
```

4. **Install dependencies & run**
```bash
go mod tidy
go run ./cmd/bot/
```

Atau menggunakan Makefile:
```bash
make run
```

### Build
```bash
make build
# Binary akan ada di ./bin/pnj-bot dan ./bin/pnj-csbot
```

## 📁 Struktur Project

```
pnj-anonymous-bot/
├── cmd/bot/main.go              # Entry point
├── internal/
│   ├── bot/
│   │   ├── bot.go               # Core bot & router
│   │   ├── handlers.go          # Command handlers
│   │   ├── callbacks.go         # Inline keyboard callbacks
│   │   └── keyboards.go         # Keyboard definitions
│   ├── config/config.go         # Environment config
│   ├── database/
│   │   ├── database.go          # DB setup & migrations
│   │   ├── user.go              # User CRUD
│   │   ├── chat.go              # Chat session operations
│   │   ├── confession.go        # Confession CRUD
│   │   └── report.go            # Reports, blocks, OTP
│   ├── email/sender.go          # Brevo email sender
│   ├── models/models.go         # Data models
│   └── service/
│       ├── auth.go              # Authentication logic
│       ├── chat.go              # Chat matching (Redis queue)
│       ├── confession.go        # Confession logic
│       └── profile.go           # Profile management
├── .env.example
├── go.mod
├── Makefile
└── README.md
```

## 🔧 Cara Membuat Bot Telegram

1. Buka [@BotFather](https://t.me/BotFather) di Telegram
2. Kirim `/newbot`
3. Ikuti instruksi untuk memberi nama bot
4. Salin token yang diberikan ke `.env`

## ?? Setup Brevo (OTP Email)

1. Buat akun di [Brevo](https://www.brevo.com/)
2. Generate API Key dari menu SMTP & API
3. Verifikasi sender email/domain kamu di Brevo
4. Isi `BREVO_API_KEY`, `SMTP_USERNAME`, dan `SMTP_FROM` di `.env`


## 🐳 Docker Deployment

Bot ini sudah dioptimalkan untuk berjalan di Docker dengan ukuran image sangat kecil (~20MB) dan mendukung arsitektur multi-stage.

### Menggunakan Helper Script (Recommended)

**Windows:**
```cmd
.\scripts\deploy.bat        # Jalankan mode development
.\scripts\deploy.bat prod   # Jalankan mode production
.\scripts\deploy.bat stop   # Hentikan semua container
.\scripts\deploy.bat logs   # Lihat log live
.\scripts\deploy.bat clean  # Hapus semua data & container
```

**Linux/Mac:**
```bash
chmod +x scripts/deploy.sh
./scripts/deploy.sh         # Jalankan mode development
./scripts/deploy.sh prod    # Jalankan mode production
```

### Manual dengan Docker Compose

**Development:**
```bash
# Otomatis build & restart
docker compose up --build -d
```

**Production:**
```bash
# Overlay mode prod (resource limits & restart policy always)
docker compose -f docker-compose.yml -f docker-compose.prod.yml up --build -d
```

### Fitur Docker:
- **Health Checks**: Endpoint `/health` terekspos di port 8080 untuk monitoring status bot.
- **Persistent Storage**: Database tersimpan aman di volume `pnj-anonymous-bot-data`.
- **Auto-Restart**: Otomatis restart jika crash (policy `unless-stopped` di dev, `always` di prod).
- **Security**: Berjalan sebagai non-root user (`pnjbot`) dengan sistem file read-only.
- **Logging**: Log rotasi otomatis agar tidak memenuhi disk.

## 📝 License

[MIT License](LICENSE) — Politeknik Negeri Jakarta © 2026



