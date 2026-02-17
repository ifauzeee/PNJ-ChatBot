package service

import (
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

func (s *ConfessionService) CreateConfession(telegramID int64, content string) (*models.Confession, error) {

	count, err := s.db.GetUserConfessionCount(telegramID, time.Now().Add(-1*time.Hour))
	if err != nil {
		return nil, err
	}
	if count >= s.cfg.MaxConfessionsPerHour {
		return nil, fmt.Errorf("kamu sudah mencapai batas %d confession per jam. Coba lagi nanti", s.cfg.MaxConfessionsPerHour)
	}

	user, err := s.db.GetUser(telegramID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found")
	}

	dept := string(user.Department)
	confession, err := s.db.CreateConfession(telegramID, content, dept)
	if err != nil {
		return nil, err
	}

	return confession, nil
}

func (s *ConfessionService) GetLatestConfessions(limit int) ([]*models.Confession, error) {
	return s.db.GetLatestConfessions(limit)
}

func (s *ConfessionService) ReactToConfession(confessionID, telegramID int64, reaction string) error {
	return s.db.AddConfessionReaction(confessionID, telegramID, reaction)
}

func (s *ConfessionService) GetReactionCounts(confessionID int64) (map[string]int, error) {
	return s.db.GetConfessionReactionCounts(confessionID)
}

func (s *ConfessionService) GetConfession(id int64) (*models.Confession, error) {
	return s.db.GetConfession(id)
}
