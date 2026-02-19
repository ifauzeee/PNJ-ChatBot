package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/pnj-anonymous-bot/internal/config"
	"github.com/pnj-anonymous-bot/internal/database"
	"github.com/pnj-anonymous-bot/internal/email"
	"github.com/pnj-anonymous-bot/internal/logger"
	"github.com/pnj-anonymous-bot/internal/metrics"
	"github.com/pnj-anonymous-bot/internal/models"
	"github.com/pnj-anonymous-bot/internal/service"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"


	"github.com/getsentry/sentry-go"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	updateQueueDepthGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pnj_bot_update_queue_depth",
		Help: "Current number of pending Telegram updates in worker queue.",
	})

	updateProcessDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "pnj_bot_update_process_duration_seconds",
		Help:    "Time spent processing a Telegram update in worker pool.",
		Buckets: prometheus.ExponentialBuckets(0.005, 2, 12),
	}, []string{"user_id", "update_type"})

	userLockWaitSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "pnj_bot_user_lock_wait_seconds",
		Help:    "Wait time to acquire per-user lock before update processing.",
		Buckets: prometheus.ExponentialBuckets(0.0005, 2, 12),
	}, []string{"user_id"})

	userLockContentionTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pnj_bot_user_lock_contention_total",
		Help: "Number of updates waiting for contested per-user locks.",
	}, []string{"user_id"})
)

type Bot struct {
	api          *tgbotapi.BotAPI
	cfg          *config.Config
	db           *database.DB
	redisSvc     *service.RedisService
	auth         *service.AuthService
	chat         *service.ChatService
	confession   *service.ConfessionService
	profile      *service.ProfileService
	room         *service.RoomService
	moderation   *service.ModerationService
	profanity    *service.ProfanityService
	evidence     *service.EvidenceService
	gamification *service.GamificationService
	startedAt    time.Time
	updateQ      chan tgbotapi.Update
	updateWG     sync.WaitGroup
	background   sync.WaitGroup
	userLocks    sync.Map
	handlers     map[string]func(context.Context, *tgbotapi.Message)
	callbacks    map[string]func(context.Context, int64, string, *tgbotapi.CallbackQuery)
}

func New(cfg *config.Config, db *database.DB) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, err
	}

	api.Debug = cfg.BotDebug

	emailSender := email.NewSender(cfg)
	redisSvc := service.NewRedisService()

	bot := &Bot{
		api:          api,
		cfg:          cfg,
		db:           db,
		redisSvc:     redisSvc,
		auth:         service.NewAuthService(db, emailSender, cfg),
		chat:         service.NewChatService(db, redisSvc, cfg.MaxSearchPerMinute),
		confession:   service.NewConfessionService(db, cfg),
		profile:      service.NewProfileService(db, cfg),
		room:         service.NewRoomService(db),
		moderation:   service.NewModerationService(cfg),
		profanity:    service.NewProfanityService(),
		evidence:     service.NewEvidenceService(db, redisSvc.GetClient()),
		gamification: service.NewGamificationService(db),
		startedAt:    time.Now(),
		updateQ:      make(chan tgbotapi.Update, cfg.MaxUpdateQueue),
	}

	bot.registerHandlers()
	logger.Info("🤖 Bot authorized", zap.String("username", api.Self.UserName))
	return bot, nil
}

func (b *Bot) registerHandlers() {
	b.handlers = map[string]func(context.Context, *tgbotapi.Message){
		"start":        b.handleStart,
		"regist":       b.handleRegist,
		"help":         b.handleHelp,
		"about":        b.handleAbout,
		"cancel":       b.handleCancel,
		"search":       b.handleSearch,
		"next":         b.handleNext,
		"stop":         b.handleStop,
		"confess":      b.handleConfess,
		"confessions":  b.handleConfessions,
		"react":        b.handleReact,
		"reply":        b.handleReply,
		"view_replies": b.handleViewReplies,
		"poll":         b.handlePoll,
		"polls":        b.handleViewPolls,
		"vote_poll":    b.handleVotePoll,
		"whisper":      b.handleWhisper,
		"profile":      b.handleProfile,
		"stats":        b.handleStats,
		"leaderboard":  b.handleLeaderboard,
		"admin_poll":   b.handleAdminPoll,
		"broadcast":    b.handleBroadcast,
		"edit":         b.handleEdit,
		"report":       b.handleReport,
		"block":        b.handleBlock,
		"circles":      b.handleCircles,
		"leave_circle": b.handleLeaveCircle,
	}

	b.callbacks = map[string]func(context.Context, int64, string, *tgbotapi.CallbackQuery){
		"gender":  b.handleGenderCallback,
		"dept":    b.handleDeptCallback,
		"search":  b.handleSearchCallback,
		"chat":    b.handleChatActionCallback,
		"menu":    b.handleMenuCallback,
		"edit":    b.handleEditCallback,
		"vote":    b.handleVoteCallback,
		"year":    b.handleYearCallback,
		"react":   b.handleReactionCallback,
		"whisper": b.handleWhisperCallback,
		"legal":   b.handleLegalCallback,
		"circle":  b.handleCircleCallback,
	}
}

type HealthResponse struct {
	Status    string `json:"status"`
	Database  string `json:"database"`
	Redis     string `json:"redis"`
	System    struct {
		UptimeSeconds int64  `json:"uptime_seconds"`
		HeapAlloc     uint64 `json:"heap_alloc"`
		StackInUse    uint64 `json:"stack_in_use"`
		Goroutines    int    `json:"goroutines"`
	} `json:"system"`
	Stats struct {
		TotalUserOnline int `json:"total_user_online"`
		TotalUserQueue  int `json:"total_user_queue"`
	} `json:"stats"`
	Timestamp string `json:"timestamp"`
}

func (b *Bot) startHealthServer(ctx context.Context) {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		ctx := r.Context()
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		userCount, _ := b.db.GetOnlineUserCount(ctx)
		queueCount, _ := b.chat.GetQueueCount(ctx)

		dbErr := b.db.PingContext(ctx)
		redisErr := b.redisSvc.Ping(ctx)

		dbStatus := "ok"
		if dbErr != nil {
			dbStatus = "error"
		}
		redisStatus := "ok"
		if redisErr != nil {
			redisStatus = "error"
		}

		health := HealthResponse{
			Status:   "ok",
			Database: dbStatus,
			Redis:    redisStatus,
			Timestamp: time.Now().Format(time.RFC3339),
		}
		health.System.UptimeSeconds = int64(time.Since(b.startedAt).Seconds())
		health.System.HeapAlloc = memStats.Alloc
		health.System.StackInUse = memStats.StackInuse
		health.System.Goroutines = runtime.NumGoroutine()
		health.Stats.TotalUserOnline = userCount
		health.Stats.TotalUserQueue = queueCount

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(health)
	})

	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()

		dbErr := b.db.PingContext(ctx)
		redisErr := b.redisSvc.Ping(ctx)

		if dbErr != nil || redisErr != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			dbStatus := "ok"
			redisStatus := "ok"
			if dbErr != nil {
				dbStatus = dbErr.Error()
			}
			if redisErr != nil {
				redisStatus = redisErr.Error()
			}
			resp := struct {
				Status   string `json:"status"`
				Database string `json:"database"`
				Redis    string `json:"redis"`
			}{
				Status:   "not_ready",
				Database: dbStatus,
				Redis:    redisStatus,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready","database":"ok","redis":"ok"}`))
	})

	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Warn("Health server shutdown error", zap.Error(err))
		}
	}()

	logger.Info("🏥 Health check server listening on :8080")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("⚠️ Health check server error", zap.Error(err))
	}
}

func (b *Bot) startQueueWorker(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			updatedIDs, err := b.chat.ProcessQueueTimeout(ctx, 60)
			if err != nil {
				logger.Error("⚠️ Queue worker error", zap.Error(err))
				continue
			}

			for _, telegramID := range updatedIDs {
				msg := `⏳ *Belum menemukan partner...*

Karena belum ada partner yang cocok dengan kriteria kamu, sekarang bot akan mencari partner secara *acak* agar lebih cepat.

_Mohon tunggu sebentar ya..._`
				b.sendMessage(telegramID, msg, nil)
			}
		}
	}
}

func (b *Bot) startUpdateWorkers() {
	for i := 0; i < b.cfg.MaxUpdateWorkers; i++ {
		workerID := i + 1
		b.updateWG.Add(1)

		go func() {
			defer b.updateWG.Done()
			for update := range b.updateQ {
				updateQueueDepthGauge.Set(float64(len(b.updateQ)))
				b.handleUpdate(update)
			}
			logger.Debug("Update worker stopped", zap.Int("worker_id", workerID))
		}()
	}

	logger.Info("Update worker pool started",
		zap.Int("workers", b.cfg.MaxUpdateWorkers),
		zap.Int("queue_size", cap(b.updateQ)),
	)
	updateQueueDepthGauge.Set(0)
}

func (b *Bot) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger.Info("🚀 Starting PNJ Anonymous Bot...")

	b.background.Add(1)
	go func() {
		defer b.background.Done()
		b.startHealthServer(runCtx)
	}()

	b.background.Add(1)
	go func() {
		defer b.background.Done()
		b.startQueueWorker(runCtx)
	}()

	b.startUpdateWorkers()

	commands := []tgbotapi.BotCommand{
		{Command: "start", Description: "🎭 Mulai bot / Menu utama"},
		{Command: "regist", Description: "📝 Registrasi akun baru"},
		{Command: "search", Description: "🔍 Cari partner chat anonim"},
		{Command: "next", Description: "⏭️ Skip ke partner berikutnya"},
		{Command: "stop", Description: "🛑 Hentikan chat saat ini"},
		{Command: "confess", Description: "💬 Kirim confession anonim"},
		{Command: "confessions", Description: "📋 Lihat confession terbaru"},
		{Command: "react", Description: "❤️ Reaksi ke confession"},
		{Command: "reply", Description: "Balas confession (contoh: /reply 1 Hallo!)"},
		{Command: "view_replies", Description: "Lihat balasan confession (contoh: /view_replies 1)"},
		{Command: "poll", Description: "🗳️ Buat polling anonim"},
		{Command: "polls", Description: "📊 Lihat daftar polling"},
		{Command: "vote_poll", Description: "🗳️ Ikut memilih polling (contoh: /vote_poll 1)"},
		{Command: "whisper", Description: "📢 Kirim whisper ke jurusan"},
		{Command: "circles", Description: "👥 Gabung Group Circle Anonim"},
		{Command: "profile", Description: "👤 Lihat profil kamu"},
		{Command: "stats", Description: "📊 Statistik kamu"},
		{Command: "leaderboard", Description: "🏆 Peringkat pengguna teraktif"},
		{Command: "edit", Description: "✏️ Edit profil"},
		{Command: "about", Description: "⚖️ Informasi hukum & disclaimer"},
		{Command: "help", Description: "❓ Bantuan & panduan"},
		{Command: "cancel", Description: "❌ Batalkan aksi saat ini"},
		{Command: "admin_poll", Description: "📢 (Admin) Buat polling global"},
		{Command: "broadcast", Description: "📢 (Admin) Broadcast pesan global"},
	}
	cmdCfg := tgbotapi.NewSetMyCommands(commands...)
	if _, err := b.api.Request(cmdCfg); err != nil {
		logger.Warn("Failed to set bot commands", zap.Error(err))
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

intakeLoop:
	for {
		select {
		case <-runCtx.Done():
			logger.Info("Stopping update intake...")
			b.api.StopReceivingUpdates()
			break intakeLoop
		case update, ok := <-updates:
			if !ok {
				break intakeLoop
			}

			select {
			case b.updateQ <- update:
				updateQueueDepthGauge.Set(float64(len(b.updateQ)))
			case <-runCtx.Done():
				logger.Info("Stopping update intake...")
				b.api.StopReceivingUpdates()
				break intakeLoop
			}
		}
	}

	close(b.updateQ)

	logger.Info("⏳ Waiting for update workers to finish...",
		zap.Int("remaining_updates", len(b.updateQ)),
	)
	workersDone := make(chan struct{})
	go func() {
		b.updateWG.Wait()
		close(workersDone)
	}()

	shutdownTimeout := 30 * time.Second
	select {
	case <-workersDone:
		logger.Info("✅ All update workers finished")
	case <-time.After(shutdownTimeout):
		logger.Warn("⚠️ Update workers did not finish within timeout",
			zap.Duration("timeout", shutdownTimeout),
		)
	}

	cancel()
	logger.Info("⏳ Waiting for background tasks to finish...")

	bgDone := make(chan struct{})
	go func() {
		b.background.Wait()
		close(bgDone)
	}()

	select {
	case <-bgDone:
		logger.Info("✅ All background tasks finished")
	case <-time.After(10 * time.Second):
		logger.Warn("⚠️ Background tasks did not finish within timeout")
	}

	updateQueueDepthGauge.Set(0)

	logger.Info("⏳ Closing external connections...")
	if err := b.redisSvc.Close(); err != nil {
		logger.Warn("Redis close error", zap.Error(err))
	}
	if err := b.db.Close(); err != nil {
		logger.Warn("Database close error", zap.Error(err))
	}
	logger.Info("✅ External connections closed")

	logger.Info("🛑 Bot shutdown completed")

	return nil
}

func (b *Bot) handleUpdate(update tgbotapi.Update) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	startedAt := time.Now()
	userID, hasUser := b.extractUpdateUserID(update)
	userLabel := updateMetricUserLabel(userID, hasUser)
	updateType := classifyUpdate(update)

	defer func() {
		if r := recover(); r != nil {
			logger.Error("❌ Panic recovered", zap.Any("recover", r))
			sentry.CurrentHub().Recover(r)
		}
		updateProcessDurationSeconds.WithLabelValues(userLabel, updateType).Observe(time.Since(startedAt).Seconds())
	}()

	if hasUser {
		lock := b.getUserLock(userID)
		lockWaitStart := time.Now()
		lock.Lock()
		lockWait := time.Since(lockWaitStart)
		userLockWaitSeconds.WithLabelValues(userLabel).Observe(lockWait.Seconds())
		if lockWait > time.Millisecond {
			userLockContentionTotal.WithLabelValues(userLabel).Inc()
		}
		defer lock.Unlock()

		streak, bonus, errStreak := b.gamification.UpdateStreak(ctx, userID)
		if errStreak == nil {
			if bonus {
				_, _, _, _, _ = b.gamification.RewardActivity(ctx, userID, "streak_bonus")
				b.sendMessageHTML(userID, fmt.Sprintf("🔥 <b>STREAK LANJUT!</b>\nKamu sudah aktif selama <b>%d hari</b> berturut-turut! Dapat bonus poin dan exp.", streak), nil)
			} else {
				_, _, _, _, _ = b.gamification.RewardActivity(ctx, userID, "daily_login")
			}
		}
	}

	if update.CallbackQuery != nil {
		b.handleCallback(ctx, update.CallbackQuery)
		return
	}

	if update.Message == nil {
		return
	}

	if update.Message.IsCommand() {
		b.handleCommand(ctx, update.Message)
		return
	}

	b.handleMessage(ctx, update.Message)
}

func updateMetricUserLabel(userID int64, hasUser bool) string {
	if !hasUser {
		return "unknown"
	}
	return strconv.FormatInt(userID, 10)
}

func classifyUpdate(update tgbotapi.Update) string {
	if update.CallbackQuery != nil {
		return "callback"
	}
	if update.Message == nil {
		return "other"
	}
	if update.Message.IsCommand() {
		return "command"
	}
	return "message"
}

func (b *Bot) extractUpdateUserID(update tgbotapi.Update) (int64, bool) {
	if update.CallbackQuery != nil && update.CallbackQuery.From != nil {
		return update.CallbackQuery.From.ID, true
	}
	if update.Message != nil && update.Message.From != nil {
		return update.Message.From.ID, true
	}
	return 0, false
}

func (b *Bot) getUserLock(userID int64) *sync.Mutex {
	if lock, ok := b.userLocks.Load(userID); ok {
		return lock.(*sync.Mutex)
	}

	newLock := &sync.Mutex{}
	actual, _ := b.userLocks.LoadOrStore(userID, newLock)
	return actual.(*sync.Mutex)
}

func (b *Bot) handleCommand(ctx context.Context, msg *tgbotapi.Message) {
	telegramID := msg.From.ID
	command := msg.Command()

	handler, exists := b.handlers[command]
	if !exists {
		b.sendMessage(telegramID, "❓ Perintah tidak dikenali. Ketik /help untuk bantuan.", nil)
		return
	}

	metrics.CommandsTotal.WithLabelValues(command).Inc()

	if command == "start" || command == "help" || command == "about" || command == "cancel" || command == "regist" {
		handler(ctx, msg)
		return
	}

	if !b.requireVerification(ctx, msg) {
		return
	}

	if banned, _ := b.auth.IsBanned(ctx, telegramID); banned {
		b.sendMessage(telegramID, "🚫 *Akun kamu telah di-banned.*\n\nKamu tidak bisa menggunakan bot ini karena telah melanggar aturan.", nil)
		return
	}

	handler(ctx, msg)
}

func (b *Bot) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	state, stateData, err := b.db.GetUserState(ctx, telegramID)
	if err != nil {
		logger.Error("Error getting user state", zap.Int64("telegram_id", telegramID), zap.Error(err))
		return
	}

	switch state {
	case models.StateAwaitingEmail:
		b.handleEmailInput(ctx, msg)
	case models.StateAwaitingOTP:
		b.handleOTPInput(ctx, msg)
	case models.StateInChat:
		b.handleChatMessage(ctx, msg)
	case models.StateAwaitingConfess:
		b.handleConfessionInput(ctx, msg)
	case models.StateAwaitingReport:
		b.handleReportInput(ctx, msg)
	case models.StateAwaitingWhisper:
		b.handleWhisperInput(ctx, msg, stateData)
	case models.StateInCircle:
		b.handleCircleMessage(ctx, msg)
	case models.StateAwaitingRoomName:
		b.handleRoomNameInput(ctx, msg)
	case models.StateAwaitingRoomDesc:
		b.handleRoomDescInput(ctx, msg)
	default:

		if msg.Text != "" {
			b.sendMessage(telegramID, "💡 Gunakan /start untuk membuka menu utama atau /help untuk bantuan.", nil)
		}
	}
}

func (b *Bot) requireVerification(ctx context.Context, msg *tgbotapi.Message) bool {
	telegramID := msg.From.ID

	if b.cfg.MaintenanceAccountID != 0 && telegramID == b.cfg.MaintenanceAccountID {
		user, _ := b.db.GetUser(ctx, telegramID)
		if user == nil {
			if _, err := b.db.CreateUser(ctx, telegramID); err != nil {
				logIfErr("create_maintenance_user", err)
			}
			logIfErr("set_maintenance_display_name", b.db.UpdateUserDisplayName(ctx, telegramID, "🛠️ Maintenance Account"))
			logIfErr("set_maintenance_verified", b.db.UpdateUserVerified(ctx, telegramID, true))
			logIfErr("set_maintenance_gender", b.db.UpdateUserGender(ctx, telegramID, "Maintenance"))
			logIfErr("set_maintenance_dept", b.db.UpdateUserDepartment(ctx, telegramID, "System"))
		}
		return true
	}

	user, err := b.db.GetUser(ctx, telegramID)
	if err != nil || user == nil {
		b.sendMessage(telegramID, "⚠️ Kamu belum terdaftar. Ketik /start untuk memulai.", nil)
		return false
	}

	if !user.IsVerified {
		b.sendMessage(telegramID, "⚠️ *Email belum diverifikasi!*\n\nKetik /regist dan ikuti proses verifikasi email PNJ kamu.", nil)
		return false
	}

	if string(user.Gender) == "" || string(user.Department) == "" {
		b.sendMessage(telegramID, "⚠️ *Profil belum lengkap!*\n\nKetik /start untuk melengkapi profil kamu.", nil)
		return false
	}

	return true
}

func (b *Bot) sendMessage(chatID int64, text string, keyboard *tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	if keyboard != nil {
		msg.ReplyMarkup = keyboard
	}

	if _, err := b.api.Send(msg); err != nil {
		logger.Error("Error sending message", zap.Int64("chat_id", chatID), zap.Error(err))
	}
}

func (b *Bot) sendMessageHTML(chatID int64, text string, keyboard *tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	if keyboard != nil {
		msg.ReplyMarkup = keyboard
	}

	if _, err := b.api.Send(msg); err != nil {
		logger.Error("Error sending HTML message", zap.Int64("chat_id", chatID), zap.Error(err))
	}
}

func (b *Bot) answerCallback(callbackID string, text string) {
	callback := tgbotapi.NewCallback(callbackID, text)
	if _, err := b.api.Request(callback); err != nil {
		logger.Error("Error answering callback", zap.String("callback_id", callbackID), zap.Error(err))
	}
}
