package service

import (
	"context"

	"github.com/pnj-anonymous-bot/internal/database"
)

type CSService struct {
	db *database.DB
}

func NewCSService(db *database.DB) *CSService {
	return &CSService{
		db: db,
	}
}

func (s *CSService) GetTimedOutSessions(ctx context.Context, timeoutMinutes int) ([]int64, error) {
	return s.db.GetTimedOutCSSessions(ctx, timeoutMinutes)
}

func (s *CSService) GetActiveSessionByAdmin(ctx context.Context, adminID int64) (int64, error) {
	return s.db.GetActiveCSSessionByAdmin(ctx, adminID)
}

func (s *CSService) GetActiveSessionByUser(ctx context.Context, userID int64) (int64, error) {
	return s.db.GetActiveCSSessionByUser(ctx, userID)
}

func (s *CSService) UpdateSessionActivity(ctx context.Context, userID int64) error {
	return s.db.UpdateCSSessionActivity(ctx, userID)
}

func (s *CSService) JoinQueue(ctx context.Context, userID int64) error {
	return s.db.JoinCSQueue(ctx, userID)
}

func (s *CSService) GetQueuePosition(ctx context.Context, userID int64) (int, error) {
	return s.db.GetCSQueuePosition(ctx, userID)
}

func (s *CSService) LeaveQueue(ctx context.Context, userID int64) error {
	return s.db.LeaveCSQueue(ctx, userID)
}

func (s *CSService) CreateSession(ctx context.Context, userID, adminID int64) error {
	return s.db.CreateCSSession(ctx, userID, adminID)
}

func (s *CSService) EndSession(ctx context.Context, userID int64) error {
	return s.db.EndCSSession(ctx, userID)
}

func (s *CSService) GetNextInQueue(ctx context.Context) (int64, error) {
	return s.db.GetNextInCSQueue(ctx)
}
