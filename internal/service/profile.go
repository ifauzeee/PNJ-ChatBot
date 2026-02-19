package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/pnj-anonymous-bot/internal/config"
	"github.com/pnj-anonymous-bot/internal/database"
	"github.com/pnj-anonymous-bot/internal/logger"
	"github.com/pnj-anonymous-bot/internal/models"
	"go.uber.org/zap"
)

type ProfileService struct {
	db  *database.DB
	cfg *config.Config
}

func NewProfileService(db *database.DB, cfg *config.Config) *ProfileService {
	return &ProfileService{db: db, cfg: cfg}
}

func (s *ProfileService) SetGender(ctx context.Context, telegramID int64, gender string) error {
	if !models.IsValidGender(gender) {
		return fmt.Errorf("gender tidak valid")
	}

	if err := s.db.UpdateUserGender(ctx, telegramID, gender); err != nil {
		return err
	}

	return s.db.SetUserState(ctx, telegramID, models.StateAwaitingYear, "")
}

func (s *ProfileService) SetYear(ctx context.Context, telegramID int64, year int) error {
	if !models.IsValidEntryYear(year) {
		return fmt.Errorf("tahun angkatan tidak valid")
	}

	if err := s.db.UpdateUserYear(ctx, telegramID, year); err != nil {
		return err
	}

	return s.db.SetUserState(ctx, telegramID, models.StateAwaitingDept, "")
}

func (s *ProfileService) SetDepartment(ctx context.Context, telegramID int64, dept string) error {
	if !models.IsValidDepartment(dept) {
		return fmt.Errorf("jurusan tidak valid")
	}

	if err := s.db.UpdateUserDepartment(ctx, telegramID, dept); err != nil {
		return err
	}

	displayName := generateDisplayName()
	if err := s.db.UpdateUserDisplayName(ctx, telegramID, displayName); err != nil {
		return err
	}

	return s.db.SetUserState(ctx, telegramID, models.StateNone, "")
}

func (s *ProfileService) GetProfile(ctx context.Context, telegramID int64) (*models.User, error) {
	return s.db.GetUser(ctx, telegramID)
}

func (s *ProfileService) GetStats(ctx context.Context, telegramID int64) (totalChats, totalConfessions, totalReactions, daysSinceJoined int, err error) {
	return s.db.GetUserStats(ctx, telegramID)
}

func (s *ProfileService) UpdateGender(ctx context.Context, telegramID int64, gender string) error {
	if !models.IsValidGender(gender) {
		return fmt.Errorf("gender tidak valid")
	}
	return s.db.UpdateUserGender(ctx, telegramID, gender)
}

func (s *ProfileService) UpdateYear(ctx context.Context, telegramID int64, year int) error {
	if !models.IsValidEntryYear(year) {
		return fmt.Errorf("tahun angkatan tidak valid")
	}
	return s.db.UpdateUserYear(ctx, telegramID, year)
}

func (s *ProfileService) UpdateDepartment(ctx context.Context, telegramID int64, dept string) error {
	if !models.IsValidDepartment(dept) {
		return fmt.Errorf("jurusan tidak valid")
	}
	return s.db.UpdateUserDepartment(ctx, telegramID, dept)
}

func (s *ProfileService) ReportUser(ctx context.Context, reporterID, reportedID int64, reason, evidence string, chatSessionID int64) (int, error) {
	count, err := s.db.GetUserReportCount(ctx, reporterID, time.Now().Add(-24*time.Hour))
	if err != nil {
		return 0, err
	}
	if count >= s.cfg.MaxReportsPerDay {
		return 0, fmt.Errorf("kamu sudah mencapai batas laporan per hari")
	}

	if err := s.db.CreateReport(ctx, reporterID, reportedID, reason, evidence, chatSessionID); err != nil {
		return 0, err
	}

	newCount, err := s.db.IncrementReportCount(ctx, reportedID)
	if err != nil {
		return 0, err
	}

	if newCount >= s.cfg.AutoBanReportCount {
		if err := s.db.UpdateUserBanned(ctx, reportedID, true); err != nil {
			logger.Warn("Failed to auto-ban reported user",
				zap.Int64("reported_id", reportedID),
				zap.Error(err),
			)
		}
	}

	return newCount, nil
}

func (s *ProfileService) BlockUser(ctx context.Context, userID, blockedID int64) error {
	return s.db.BlockUser(ctx, userID, blockedID)
}

func (s *ProfileService) SendWhisper(ctx context.Context, senderID int64, targetDept, content string) ([]int64, error) {
	user, err := s.db.GetUser(ctx, senderID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found")
	}

	if s.cfg.MaxWhispersPerHour > 0 {
		count, err := s.db.GetUserWhisperCount(ctx, senderID, time.Now().Add(-1*time.Hour))
		if err == nil && count >= s.cfg.MaxWhispersPerHour {
			return nil, fmt.Errorf("kamu sudah mencapai batas %d whisper per jam. Coba lagi nanti", s.cfg.MaxWhispersPerHour)
		}
	}

	_, err = s.db.CreateWhisper(ctx, senderID, targetDept, content, string(user.Department), string(user.Gender))
	if err != nil {
		return nil, err
	}

	targets, err := s.db.GetUsersByDepartment(ctx, targetDept, senderID)
	if err != nil {
		return nil, err
	}

	return targets, nil
}

var (
	adjectives = []string{
		"Mysterious", "Silent", "Shadow", "Hidden", "Phantom",
		"Secret", "Unknown", "Masked", "Invisible", "Anonymous",
		"Cosmic", "Stellar", "Neon", "Cyber", "Digital",
		"Mystic", "Dark", "Light", "Swift", "Bold",
	}

	animals = []string{
		"Fox", "Wolf", "Eagle", "Owl", "Tiger",
		"Panther", "Hawk", "Raven", "Phoenix", "Dragon",
		"Falcon", "Bear", "Lion", "Deer", "Cobra",
		"Jaguar", "Lynx", "Viper", "Crane", "Shark",
	}
)

func generateDisplayName() string {
	adj := adjectives[getSecureRandomInt(len(adjectives))]
	animal := animals[getSecureRandomInt(len(animals))]
	num := getSecureRandomInt(999) + 1
	return fmt.Sprintf("%s%s%d", adj, animal, num)
}

func getSecureRandomInt(max int) int {
	if max <= 0 {
		return 0
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	return int(n.Int64())
}
