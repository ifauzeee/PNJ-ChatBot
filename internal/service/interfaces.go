package service

import (
	"context"

	"github.com/pnj-anonymous-bot/internal/models"
)

type Authenticator interface {
	RegisterUser(ctx context.Context, telegramID int64) (*models.User, error)
	InitiateVerification(ctx context.Context, telegramID int64, emailAddr string) error
	VerifyOTP(ctx context.Context, telegramID int64, code string) (bool, error)
	IsVerified(ctx context.Context, telegramID int64) (bool, error)
	IsProfileComplete(ctx context.Context, telegramID int64) (bool, error)
	IsBanned(ctx context.Context, telegramID int64) (bool, error)
}

type ChatManager interface {
	SearchPartner(ctx context.Context, telegramID int64, preferredDept, preferredGender string, preferredYear int) (int64, error)
	StopChat(ctx context.Context, telegramID int64) (int64, error)
	NextPartner(ctx context.Context, telegramID int64) (int64, error)
	GetPartner(ctx context.Context, telegramID int64) (int64, error)
	GetPartnerInfo(ctx context.Context, partnerID int64) (string, string, int, error)
	GetQueueCount(ctx context.Context) (int, error)
	CancelSearch(ctx context.Context, telegramID int64) error
	ProcessQueueTimeout(ctx context.Context, timeoutSeconds int) ([]int64, error)
}

type ConfessionManager interface {
	CreateConfession(ctx context.Context, telegramID int64, content string) (*models.Confession, error)
	GetLatestConfessions(ctx context.Context, limit int) ([]*models.Confession, error)
	ReactToConfession(ctx context.Context, confessionID, telegramID int64, reaction string) error
	GetReactionCounts(ctx context.Context, confessionID int64) (map[string]int, error)
	GetConfession(ctx context.Context, id int64) (*models.Confession, error)
}

type ProfileManager interface {
	SetGender(ctx context.Context, telegramID int64, gender string) error
	SetYear(ctx context.Context, telegramID int64, year int) error
	SetDepartment(ctx context.Context, telegramID int64, dept string) error
	GetProfile(ctx context.Context, telegramID int64) (*models.User, error)
	GetStats(ctx context.Context, telegramID int64) (totalChats, totalConfessions, totalReactions, daysSinceJoined int, err error)
	UpdateGender(ctx context.Context, telegramID int64, gender string) error
	UpdateYear(ctx context.Context, telegramID int64, year int) error
	UpdateDepartment(ctx context.Context, telegramID int64, dept string) error
	ReportUser(ctx context.Context, reporterID, reportedID int64, reason, evidence string, chatSessionID int64) (int, error)
	BlockUser(ctx context.Context, userID, blockedID int64) error
	SendWhisper(ctx context.Context, senderID int64, targetDept, content string) ([]int64, error)
}

type RoomManager interface {
	GetActiveRooms(ctx context.Context) ([]*models.Room, error)
	CreateRoom(ctx context.Context, name, description string) (*models.Room, error)
	JoinRoom(ctx context.Context, telegramID int64, slug string) (*models.Room, error)
	LeaveRoom(ctx context.Context, telegramID int64) error
	GetRoomMembers(ctx context.Context, telegramID int64) ([]int64, string, error)
	GetUserRoom(ctx context.Context, telegramID int64) (*models.Room, error)
}

type ContentModerator interface {
	IsSafe(ctx context.Context, imageURL string) (bool, string, error)
	IsEnabled() bool
}

type ProfanityChecker interface {
	IsBad(text string) bool
	Clean(text string) string
}

type EvidenceLogger interface {
	LogMessage(ctx context.Context, sessionID int64, senderID int64, content string, msgType string)
	GetEvidence(ctx context.Context, sessionID int64) (string, error)
	ClearEvidence(ctx context.Context, sessionID int64)
}

type Gamifier interface {
	RewardActivity(ctx context.Context, telegramID int64, activityType string) (level int, leveledUp bool, pointsEarned int, expEarned int, err error)
	UpdateStreak(ctx context.Context, telegramID int64) (newStreak int, bonus bool, err error)
	GetLeaderboard(ctx context.Context) ([]models.User, error)
}
