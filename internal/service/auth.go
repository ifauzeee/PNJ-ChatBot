package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pnj-anonymous-bot/internal/config"
	"github.com/pnj-anonymous-bot/internal/database"
	"github.com/pnj-anonymous-bot/internal/email"
	"github.com/pnj-anonymous-bot/internal/logger"
	"github.com/pnj-anonymous-bot/internal/metrics"
	"github.com/pnj-anonymous-bot/internal/models"
	"go.uber.org/zap"
)

type EmailSender interface {
	SendOTP(to, code string) error
}

type otpAttempt struct {
	count    int
	lockedAt time.Time
}

const (
	maxOTPAttempts  = 5
	otpLockDuration = 15 * time.Minute
)

type AuthService struct {
	db          *database.DB
	email       EmailSender
	cfg         *config.Config
	otpAttempts sync.Map
}

func NewAuthService(db *database.DB, emailSender EmailSender, cfg *config.Config) *AuthService {
	return &AuthService{
		db:    db,
		email: emailSender,
		cfg:   cfg,
	}
}

func (s *AuthService) RegisterUser(ctx context.Context, telegramID int64) (*models.User, error) {
	user, err := s.db.GetUser(ctx, telegramID)
	if err != nil {
		return nil, err
	}
	if user != nil {
		return user, nil
	}

	newUser, err := s.db.CreateUser(ctx, telegramID)
	if err != nil {
		return nil, err
	}
	metrics.RegistrationsTotal.Inc()
	return newUser, nil
}

func (s *AuthService) InitiateVerification(ctx context.Context, telegramID int64, emailAddr string) error {
	emailAddr = strings.ToLower(strings.TrimSpace(emailAddr))

	if !email.IsValidPNJEmail(emailAddr) {
		return fmt.Errorf("email harus menggunakan domain @mhsw.pnj.ac.id atau @stu.pnj.ac.id")
	}

	code := email.GenerateOTP(s.cfg.OTPLength)

	expiresAt := time.Now().Add(time.Duration(s.cfg.OTPExpiryMinutes) * time.Minute)
	if err := s.db.SaveVerificationCode(ctx, telegramID, emailAddr, code, expiresAt); err != nil {
		return fmt.Errorf("gagal menyimpan kode verifikasi: %w", err)
	}

	if err := s.db.UpdateUserEmail(ctx, telegramID, emailAddr); err != nil {
		return fmt.Errorf("gagal mengupdate email: %w", err)
	}

	if err := s.db.SetUserState(ctx, telegramID, models.StateAwaitingOTP, emailAddr); err != nil {
		return fmt.Errorf("gagal mengupdate state: %w", err)
	}

	if err := s.email.SendOTP(emailAddr, code); err != nil {
		logger.Error("❌ Failed to send OTP email",
			zap.String("email", emailAddr),
			zap.Error(err),
		)
		return fmt.Errorf("gagal mengirim email verifikasi. Pastikan email kamu valid dan coba lagi")
	}

	logger.Info("📧 OTP sent",
		zap.String("email", emailAddr),
		zap.Int64("user_id", telegramID),
	)
	return nil
}

func (s *AuthService) VerifyOTP(ctx context.Context, telegramID int64, code string) (bool, error) {
	code = strings.TrimSpace(code)

	if locked, remaining := s.isOTPLocked(telegramID); locked {
		return false, fmt.Errorf("terlalu banyak percobaan gagal. Coba lagi dalam %d menit", int(remaining.Minutes())+1)
	}

	verifiedEmail, valid, err := s.db.VerifyCode(ctx, telegramID, code)
	if err != nil {
		return false, err
	}
	if !valid {
		s.recordOTPFailure(telegramID)
		return false, nil
	}

	s.otpAttempts.Delete(telegramID)

	if err := s.db.UpdateUserVerified(ctx, telegramID, true); err != nil {
		return false, err
	}

	if err := s.db.UpdateUserEmail(ctx, telegramID, verifiedEmail); err != nil {
		return false, err
	}

	if err := s.db.SetUserState(ctx, telegramID, models.StateAwaitingGender, ""); err != nil {
		return false, err
	}

	metrics.VerificationsTotal.Inc()
	logger.Info("✅ User verified",
		zap.Int64("user_id", telegramID),
		zap.String("email", verifiedEmail),
	)
	return true, nil
}

func (s *AuthService) isOTPLocked(telegramID int64) (bool, time.Duration) {
	val, ok := s.otpAttempts.Load(telegramID)
	if !ok {
		return false, 0
	}
	a := val.(*otpAttempt)
	if a.count >= maxOTPAttempts && !a.lockedAt.IsZero() {
		remaining := otpLockDuration - time.Since(a.lockedAt)
		if remaining > 0 {
			return true, remaining
		}

		s.otpAttempts.Delete(telegramID)
	}
	return false, 0
}

func (s *AuthService) recordOTPFailure(telegramID int64) {
	val, _ := s.otpAttempts.LoadOrStore(telegramID, &otpAttempt{})
	a := val.(*otpAttempt)
	a.count++
	if a.count >= maxOTPAttempts {
		a.lockedAt = time.Now()
		logger.Warn("OTP brute-force lockout triggered",
			zap.Int64("user_id", telegramID),
			zap.Int("attempts", a.count),
		)
	}
}

func (s *AuthService) IsVerified(ctx context.Context, telegramID int64) (bool, error) {
	user, err := s.db.GetUser(ctx, telegramID)
	if err != nil || user == nil {
		return false, err
	}
	return user.IsVerified, nil
}

func (s *AuthService) IsProfileComplete(ctx context.Context, telegramID int64) (bool, error) {
	return s.db.IsUserProfileComplete(ctx, telegramID)
}

func (s *AuthService) IsBanned(ctx context.Context, telegramID int64) (bool, error) {
	user, err := s.db.GetUser(ctx, telegramID)
	if err != nil || user == nil {
		return false, err
	}
	return user.IsBanned, nil
}
