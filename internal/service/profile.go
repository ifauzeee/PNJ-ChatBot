package service

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/pnj-anonymous-bot/internal/config"
	"github.com/pnj-anonymous-bot/internal/database"
	"github.com/pnj-anonymous-bot/internal/models"
)

type ProfileService struct {
	db  *database.DB
	cfg *config.Config
}

func NewProfileService(db *database.DB, cfg *config.Config) *ProfileService {
	return &ProfileService{db: db, cfg: cfg}
}

func (s *ProfileService) SetGender(telegramID int64, gender string) error {
	if !models.IsValidGender(gender) {
		return fmt.Errorf("gender tidak valid")
	}

	if err := s.db.UpdateUserGender(telegramID, gender); err != nil {
		return err
	}

	return s.db.SetUserState(telegramID, models.StateAwaitingYear, "")
}

func (s *ProfileService) SetYear(telegramID int64, year int) error {
	if !models.IsValidEntryYear(year) {
		return fmt.Errorf("tahun angkatan tidak valid")
	}

	if err := s.db.UpdateUserYear(telegramID, year); err != nil {
		return err
	}

	return s.db.SetUserState(telegramID, models.StateAwaitingDept, "")
}

func (s *ProfileService) SetDepartment(telegramID int64, dept string) error {
	if !models.IsValidDepartment(dept) {
		return fmt.Errorf("jurusan tidak valid")
	}

	if err := s.db.UpdateUserDepartment(telegramID, dept); err != nil {
		return err
	}

	displayName := generateDisplayName()
	if err := s.db.UpdateUserDisplayName(telegramID, displayName); err != nil {
		return err
	}

	return s.db.SetUserState(telegramID, models.StateNone, "")
}

func (s *ProfileService) GetProfile(telegramID int64) (*models.User, error) {
	return s.db.GetUser(telegramID)
}

func (s *ProfileService) GetStats(telegramID int64) (totalChats, totalConfessions, totalReactions, daysSinceJoined int, err error) {
	return s.db.GetUserStats(telegramID)
}

func (s *ProfileService) UpdateGender(telegramID int64, gender string) error {
	if !models.IsValidGender(gender) {
		return fmt.Errorf("gender tidak valid")
	}
	return s.db.UpdateUserGender(telegramID, gender)
}

func (s *ProfileService) UpdateYear(telegramID int64, year int) error {
	if !models.IsValidEntryYear(year) {
		return fmt.Errorf("tahun angkatan tidak valid")
	}
	return s.db.UpdateUserYear(telegramID, year)
}

func (s *ProfileService) UpdateDepartment(telegramID int64, dept string) error {
	if !models.IsValidDepartment(dept) {
		return fmt.Errorf("jurusan tidak valid")
	}
	return s.db.UpdateUserDepartment(telegramID, dept)
}

func (s *ProfileService) ReportUser(reporterID, reportedID int64, reason string, chatSessionID int64) (int, error) {

	count, err := s.db.GetUserReportCount(reporterID, time.Now().Add(-24*time.Hour))
	if err != nil {
		return 0, err
	}
	if count >= s.cfg.MaxReportsPerDay {
		return 0, fmt.Errorf("kamu sudah mencapai batas laporan per hari")
	}

	if err := s.db.CreateReport(reporterID, reportedID, reason, chatSessionID); err != nil {
		return 0, err
	}

	newCount, err := s.db.IncrementReportCount(reportedID)
	if err != nil {
		return 0, err
	}

	if newCount >= s.cfg.AutoBanReportCount {
		s.db.UpdateUserBanned(reportedID, true)
	}

	return newCount, nil
}

func (s *ProfileService) BlockUser(userID, blockedID int64) error {
	return s.db.BlockUser(userID, blockedID)
}

func (s *ProfileService) SendWhisper(senderID int64, targetDept, content string) ([]int64, error) {
	user, err := s.db.GetUser(senderID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found")
	}

	_, err = s.db.CreateWhisper(senderID, targetDept, content, string(user.Department), string(user.Gender))
	if err != nil {
		return nil, err
	}

	targets, err := s.db.GetUsersByDepartment(targetDept, senderID)
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
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	adj := adjectives[r.Intn(len(adjectives))]
	animal := animals[r.Intn(len(animals))]
	num := r.Intn(999) + 1
	return fmt.Sprintf("%s%s%d", adj, animal, num)
}
