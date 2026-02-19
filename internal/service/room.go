package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/pnj-anonymous-bot/internal/database"
	"github.com/pnj-anonymous-bot/internal/logger"
	"github.com/pnj-anonymous-bot/internal/models"
	"go.uber.org/zap"
)

type RoomService struct {
	db *database.DB
}

func NewRoomService(db *database.DB) *RoomService {
	return &RoomService{db: db}
}

func (s *RoomService) GetActiveRooms(ctx context.Context) ([]*models.Room, error) {
	return s.db.GetActiveRooms(ctx)
}

func (s *RoomService) CreateRoom(ctx context.Context, name, description string) (*models.Room, error) {
	slug := s.createSlug(name)
	if slug == "" {
		return nil, fmt.Errorf("nama circle tidak valid")
	}

	existing, _ := s.db.GetRoomBySlug(ctx, slug)
	if existing != nil {
		return nil, fmt.Errorf("circle dengan nama serupa sudah ada")
	}

	return s.db.CreateRoom(ctx, slug, name, description)
}

func (s *RoomService) createSlug(name string) string {
	name = strings.ToLower(name)
	reg := regexp.MustCompile("[^a-z0-9]+")
	slug := reg.ReplaceAllString(name, "-")
	return strings.Trim(slug, "-")
}

func (s *RoomService) JoinRoom(ctx context.Context, telegramID int64, slug string) (*models.Room, error) {
	room, err := s.db.GetRoomBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, fmt.Errorf("circle tidak ditemukan")
	}

	if err := s.db.RemoveMemberFromAllRooms(ctx, telegramID); err != nil {
		logger.Warn("Failed to remove user from existing rooms before join",
			zap.Int64("user_id", telegramID),
			zap.Error(err),
		)
	}

	err = s.db.AddRoomMember(ctx, room.ID, telegramID)
	if err != nil {
		return nil, err
	}

	err = s.db.SetUserState(ctx, telegramID, models.StateInCircle, slug)
	if err != nil {
		return nil, err
	}

	logger.Debug("User joined circle",
		zap.Int64("user_id", telegramID),
		zap.String("slug", slug),
	)
	return room, nil
}

func (s *RoomService) LeaveRoom(ctx context.Context, telegramID int64) error {
	err := s.db.RemoveMemberFromAllRooms(ctx, telegramID)
	if err != nil {
		return err
	}

	err = s.db.SetUserState(ctx, telegramID, models.StateNone, "")
	if err != nil {
		return err
	}

	logger.Debug("User left circle", zap.Int64("user_id", telegramID))
	return nil
}

func (s *RoomService) GetRoomMembers(ctx context.Context, telegramID int64) ([]int64, string, error) {
	room, err := s.db.GetUserRoom(ctx, telegramID)
	if err != nil {
		return nil, "", err
	}
	if room == nil {
		return nil, "", fmt.Errorf("kamu tidak sedang berada di circle mana pun")
	}

	members, err := s.db.GetRoomMembers(ctx, room.ID)
	if err != nil {
		return nil, "", err
	}

	return members, room.Name, nil
}

func (s *RoomService) GetUserRoom(ctx context.Context, telegramID int64) (*models.Room, error) {
	return s.db.GetUserRoom(ctx, telegramID)
}
