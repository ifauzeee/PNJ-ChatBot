package service

import (
	"fmt"
	"log"

	"github.com/pnj-anonymous-bot/internal/database"
	"github.com/pnj-anonymous-bot/internal/models"
)

type ChatService struct {
	db *database.DB
}

func NewChatService(db *database.DB) *ChatService {
	return &ChatService{db: db}
}

func (s *ChatService) SearchPartner(telegramID int64, preferredDept, preferredGender string, preferredYear int) (int64, error) {

	session, err := s.db.GetActiveSession(telegramID)
	if err != nil {
		return 0, fmt.Errorf("gagal memeriksa sesi: %w", err)
	}
	if session != nil {
		return 0, fmt.Errorf("kamu masih dalam sesi chat. Gunakan /stop untuk menghentikan chat saat ini")
	}

	inQueue, err := s.db.IsInQueue(telegramID)
	if err != nil {
		return 0, err
	}
	if inQueue {
		return 0, fmt.Errorf("kamu sudah dalam antrian pencarian. Tunggu sebentar ya!")
	}

	matchID, err := s.db.FindMatch(telegramID, preferredDept, preferredGender, preferredYear)
	if err != nil {
		return 0, fmt.Errorf("gagal mencari partner: %w", err)
	}

	if matchID > 0 {

		s.db.RemoveFromQueue(matchID)
		s.db.RemoveFromQueue(telegramID)

		_, err := s.db.CreateChatSession(telegramID, matchID)
		if err != nil {
			return 0, fmt.Errorf("gagal membuat sesi chat: %w", err)
		}

		s.db.SetUserState(telegramID, models.StateInChat, "")
		s.db.SetUserState(matchID, models.StateInChat, "")

		log.Printf("💬 Chat matched: %d <-> %d", telegramID, matchID)
		return matchID, nil
	}

	if err := s.db.AddToQueue(telegramID, preferredDept, preferredGender, preferredYear); err != nil {
		return 0, fmt.Errorf("gagal menambahkan ke antrian: %w", err)
	}

	stateData := preferredDept
	if preferredGender != "" {
		if stateData != "" {
			stateData += "|" + preferredGender
		} else {
			stateData = preferredGender
		}
	}
	if preferredYear != 0 {
		yearStr := fmt.Sprintf("%d", preferredYear)
		if stateData != "" {
			stateData += "|" + yearStr
		} else {
			stateData = yearStr
		}
	}

	s.db.SetUserState(telegramID, models.StateSearching, stateData)
	log.Printf("🔍 User %d added to queue (dept: %s, gender: %s, year: %d)", telegramID, preferredDept, preferredGender, preferredYear)
	return 0, nil
}

func (s *ChatService) StopChat(telegramID int64) (int64, error) {

	s.db.RemoveFromQueue(telegramID)

	session, err := s.db.GetActiveSession(telegramID)
	if err != nil {
		return 0, err
	}
	if session == nil {
		s.db.SetUserState(telegramID, models.StateNone, "")
		return 0, nil
	}

	if err := s.db.EndChatSession(session.ID); err != nil {
		return 0, err
	}

	partnerID := session.User2ID
	if session.User1ID != telegramID {
		partnerID = session.User1ID
	}

	s.db.SetUserState(telegramID, models.StateNone, "")
	s.db.SetUserState(partnerID, models.StateNone, "")

	log.Printf("🛑 Chat ended: %d <-> %d", telegramID, partnerID)
	return partnerID, nil
}

func (s *ChatService) NextPartner(telegramID int64) (int64, error) {

	partnerID, err := s.StopChat(telegramID)
	if err != nil {
		return 0, err
	}
	return partnerID, nil
}

func (s *ChatService) GetPartner(telegramID int64) (int64, error) {
	return s.db.GetChatPartner(telegramID)
}

func (s *ChatService) GetPartnerInfo(partnerID int64) (string, string, int, error) {
	user, err := s.db.GetUser(partnerID)
	if err != nil || user == nil {
		return "", "", 0, err
	}
	return string(user.Gender), string(user.Department), user.Year, nil
}

func (s *ChatService) GetQueueCount() (int, error) {
	return s.db.GetQueueCount()
}

func (s *ChatService) CancelSearch(telegramID int64) error {
	s.db.RemoveFromQueue(telegramID)
	s.db.SetUserState(telegramID, models.StateNone, "")
	return nil
}

func (s *ChatService) ProcessQueueTimeout(timeoutSeconds int) ([]int64, error) {
	items, err := s.db.GetExpiredQueueItems(timeoutSeconds)
	if err != nil {
		return nil, err
	}

	var updatedIDs []int64
	for _, item := range items {
		if err := s.db.ClearQueueFilters(item.TelegramID); err != nil {
			log.Printf("Error clearing filters for %d: %v", item.TelegramID, err)
			continue
		}
		s.db.SetUserState(item.TelegramID, models.StateSearching, "")
		updatedIDs = append(updatedIDs, item.TelegramID)
	}

	return updatedIDs, nil
}
