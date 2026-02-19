package service

import (
	"context"
	"fmt"
	"time"

	"github.com/pnj-anonymous-bot/internal/config"
	"github.com/pnj-anonymous-bot/internal/database"
	"github.com/pnj-anonymous-bot/internal/models"
)

type ConfessionService struct {
	db  *database.DB
	cfg *config.Config
}

func NewConfessionService(db *database.DB, cfg *config.Config) *ConfessionService {
	return &ConfessionService{db: db, cfg: cfg}
}

func (s *ConfessionService) CreateConfession(ctx context.Context, telegramID int64, content string) (*models.Confession, error) {
	count, err := s.db.GetUserConfessionCount(ctx, telegramID, time.Now().Add(-1*time.Hour))
	if err != nil {
		return nil, err
	}
	if count >= s.cfg.MaxConfessionsPerHour {
		return nil, fmt.Errorf("kamu sudah mencapai batas %d confession per jam. Coba lagi nanti", s.cfg.MaxConfessionsPerHour)
	}

	user, err := s.db.GetUser(ctx, telegramID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found")
	}

	dept := string(user.Department)
	confession, err := s.db.CreateConfession(ctx, telegramID, content, dept)
	if err != nil {
		return nil, err
	}

	return confession, nil
}

func (s *ConfessionService) GetLatestConfessions(ctx context.Context, limit int) ([]*models.Confession, error) {
	return s.db.GetLatestConfessions(ctx, limit)
}

func (s *ConfessionService) ReactToConfession(ctx context.Context, confessionID, telegramID int64, reaction string) error {
	return s.db.AddConfessionReaction(ctx, confessionID, telegramID, reaction)
}

func (s *ConfessionService) GetReactionCounts(ctx context.Context, confessionID int64) (map[string]int, error) {
	return s.db.GetConfessionReactionCounts(ctx, confessionID)
}

func (s *ConfessionService) GetConfession(ctx context.Context, id int64) (*models.Confession, error) {
	return s.db.GetConfession(ctx, id)
}
