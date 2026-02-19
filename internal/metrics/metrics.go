package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	MessagesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pnj_bot_messages_total",
		Help: "Total messages processed by the bot.",
	}, []string{"type"})

	CommandsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pnj_bot_commands_total",
		Help: "Total bot commands executed.",
	}, []string{"command"})

	CallbacksTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pnj_bot_callbacks_total",
		Help: "Total callback queries handled.",
	}, []string{"category"})

	ChatMatchesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pnj_bot_chat_matches_total",
		Help: "Total successful chat partner matches.",
	})

	ChatStopsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pnj_bot_chat_stops_total",
		Help: "Total chats stopped.",
	})

	ConfessionsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pnj_bot_confessions_total",
		Help: "Total confessions created.",
	})

	ReactionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pnj_bot_reactions_total",
		Help: "Total reactions to confessions.",
	}, []string{"reaction"})

	WhispersTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pnj_bot_whispers_total",
		Help: "Total whispers sent.",
	})

	ReportsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pnj_bot_reports_total",
		Help: "Total user reports submitted.",
	})

	BlocksTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pnj_bot_blocks_total",
		Help: "Total user blocks.",
	})

	AutoBansTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pnj_bot_auto_bans_total",
		Help: "Total automatic bans triggered.",
	})

	RegistrationsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pnj_bot_registrations_total",
		Help: "Total new user registrations.",
	})

	VerificationsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pnj_bot_verifications_total",
		Help: "Total successful email verifications.",
	})

	PollsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pnj_bot_polls_total",
		Help: "Total polls created.",
	})

	VotesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pnj_bot_votes_total",
		Help: "Total poll votes.",
	})

	CircleJoinsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pnj_bot_circle_joins_total",
		Help: "Total circle joins.",
	})

	CircleLeavesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pnj_bot_circle_leaves_total",
		Help: "Total circle leaves.",
	})

	RateLimitHitsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pnj_bot_rate_limit_hits_total",
		Help: "Total rate limit hits by action.",
	}, []string{"action"})

	ProfanityFiltered = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pnj_bot_profanity_filtered_total",
		Help: "Total messages filtered for profanity.",
	})

	ModerationBlocked = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pnj_bot_moderation_blocked_total",
		Help: "Total media blocked by content moderation.",
	})

	TelegramAPIErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pnj_bot_telegram_api_errors_total",
		Help: "Total Telegram API errors.",
	}, []string{"operation"})

	DatabaseErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pnj_bot_database_errors_total",
		Help: "Total database operation errors.",
	}, []string{"operation"})

	RedisErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pnj_bot_redis_errors_total",
		Help: "Total Redis operation errors.",
	}, []string{"operation"})

	ActiveUsersGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pnj_bot_active_users",
		Help: "Number of currently active users (in chat or searching).",
	})

	QueueSizeGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pnj_bot_search_queue_size",
		Help: "Current number of users in search queue.",
	})

	BroadcastDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "pnj_bot_broadcast_duration_seconds",
		Help:    "Time spent broadcasting messages.",
		Buckets: prometheus.ExponentialBuckets(1, 2, 10),
	})
)
