package service

import (
	"fmt"

	"github.com/pnj-anonymous-bot/internal/database"
	"github.com/pnj-anonymous-bot/internal/models"
)

type GamificationService struct {
	db *database.DB
}

func NewGamificationService(db *database.DB) *GamificationService {
	return &GamificationService{db: db}
}

func (s *GamificationService) RewardActivity(telegramID int64, activityType string) (level int, leveledUp bool, pointsEarned int, expEarned int, err error) {
	switch activityType {
	case "chat_message":
		pointsEarned = 1
		expEarned = 5
	case "confession_created":
		pointsEarned = 10
		expEarned = 50
	case "reaction_given":
		pointsEarned = 2
		expEarned = 10
	case "daily_login":
		pointsEarned = 5
		expEarned = 20
	case "streak_bonus":
		pointsEarned = 15
		expEarned = 40
	default:
		return 0, false, 0, 0, fmt.Errorf("unknown activity type")
	}

	level, leveledUp, err = s.db.AddPointsAndExp(telegramID, pointsEarned, expEarned)
	return level, leveledUp, pointsEarned, expEarned, err
}

func (s *GamificationService) UpdateStreak(telegramID int64) (newStreak int, bonus bool, err error) {
	return s.db.UpdateDailyStreak(telegramID)
}

func (s *GamificationService) GetLeaderboard() ([]models.User, error) {
	return s.db.GetLeaderboard(10)
}
