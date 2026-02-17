package service

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/pnj-anonymous-bot/internal/config"
	"github.com/pnj-anonymous-bot/internal/database"
	"github.com/pnj-anonymous-bot/internal/email"
	"github.com/pnj-anonymous-bot/internal/models"
)

type AuthService struct {
	db    *database.DB
	email *email.Sender
	cfg   *config.Config
}

func NewAuthService(db *database.DB, emailSender *email.Sender, cfg *config.Config) *AuthService {
	return &AuthService{
		db:    db,
		email: emailSender,
		cfg:   cfg,
	}
}

func (s *AuthService) RegisterUser(telegramID int64) (*models.User, error) {
	user, err := s.db.GetUser(telegramID)
	if err != nil {
		return nil, err
	}
	if user != nil {
		return user, nil
	}

	return s.db.CreateUser(telegramID)
}

func (s *AuthService) InitiateVerification(telegramID int64, emailAddr string) error {
	emailAddr = strings.ToLower(strings.TrimSpace(emailAddr))

	if !email.IsValidPNJEmail(emailAddr) {
		return fmt.Errorf("email harus menggunakan domain @mhsw.pnj.ac.id atau @stu.pnj.ac.id")
	}

	code := email.GenerateOTP(s.cfg.OTPLength)

	expiresAt := time.Now().Add(time.Duration(s.cfg.OTPExpiryMinutes) * time.Minute)
	if err := s.db.SaveVerificationCode(telegramID, emailAddr, code, expiresAt); err != nil {
		return fmt.Errorf("gagal menyimpan kode verifikasi: %w", err)
	}

	if err := s.db.UpdateUserEmail(telegramID, emailAddr); err != nil {
		return fmt.Errorf("gagal mengupdate email: %w", err)
	}

	if err := s.db.SetUserState(telegramID, models.StateAwaitingOTP, emailAddr); err != nil {
		return fmt.Errorf("gagal mengupdate state: %w", err)
	}

	if err := s.email.SendOTP(emailAddr, code); err != nil {
		log.Printf("❌ Failed to send OTP email to %s: %v", emailAddr, err)
		return fmt.Errorf("gagal mengirim email verifikasi. Pastikan email kamu valid dan coba lagi")
	}

	log.Printf("📧 OTP sent to %s for user %d", emailAddr, telegramID)
	return nil
}

func (s *AuthService) VerifyOTP(telegramID int64, code string) (bool, error) {
	code = strings.TrimSpace(code)

	verifiedEmail, valid, err := s.db.VerifyCode(telegramID, code)
	if err != nil {
		return false, err
	}
	if !valid {
		return false, nil
	}

	if err := s.db.UpdateUserVerified(telegramID, true); err != nil {
		return false, err
	}

	if err := s.db.UpdateUserEmail(telegramID, verifiedEmail); err != nil {
		return false, err
	}

	if err := s.db.SetUserState(telegramID, models.StateAwaitingGender, ""); err != nil {
		return false, err
	}

	log.Printf("✅ User %d verified with email %s", telegramID, verifiedEmail)
	return true, nil
}

func (s *AuthService) IsVerified(telegramID int64) (bool, error) {
	user, err := s.db.GetUser(telegramID)
	if err != nil || user == nil {
		return false, err
	}
	return user.IsVerified, nil
}

func (s *AuthService) IsProfileComplete(telegramID int64) (bool, error) {
	return s.db.IsUserProfileComplete(telegramID)
}

func (s *AuthService) IsBanned(telegramID int64) (bool, error) {
	user, err := s.db.GetUser(telegramID)
	if err != nil || user == nil {
		return false, err
	}
	return user.IsBanned, nil
}
