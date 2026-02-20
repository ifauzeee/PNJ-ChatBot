package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pnj-anonymous-bot/internal/config"
	"github.com/pnj-anonymous-bot/internal/database"
	"github.com/pnj-anonymous-bot/internal/email"
	"github.com/pnj-anonymous-bot/internal/logger"
	"github.com/pnj-anonymous-bot/internal/metrics"
	"github.com/pnj-anonymous-bot/internal/models"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type EmailSender interface {
	SendOTP(ctx context.Context, to, code string) error
}

const (
	maxOTPAttempts  = 5
	otpLockDuration = 15 * time.Minute
)

type AuthService struct {
	db    *database.DB
	email EmailSender
	cfg   *config.Config
	redis *redis.Client
}

func NewAuthService(db *database.DB, emailSender EmailSender, cfg *config.Config, redisClient *redis.Client) *AuthService {
	return &AuthService{
		db:    db,
		email: emailSender,
		cfg:   cfg,
		redis: redisClient,
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

	if err := s.email.SendOTP(ctx, emailAddr, code); err != nil {
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

	if locked, remaining := s.isOTPLocked(ctx, telegramID); locked {
		return false, fmt.Errorf("terlalu banyak percobaan gagal. Coba lagi dalam %d menit", int(remaining.Minutes())+1)
	}

	verifiedEmail, valid, err := s.db.VerifyCode(ctx, telegramID, code)
	if err != nil {
		return false, err
	}
	if !valid {
		s.recordOTPFailure(ctx, telegramID)
		return false, nil
	}

	s.clearOTPLock(ctx, telegramID)

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

func (s *AuthService) isOTPLocked(ctx context.Context, telegramID int64) (bool, time.Duration) {
	key := fmt.Sprintf("otp_lockout:%d", telegramID)
	ttl, err := s.redis.TTL(ctx, key).Result()
	if err == nil && ttl > 0 {
		return true, ttl
	}
	return false, 0
}

func (s *AuthService) recordOTPFailure(ctx context.Context, telegramID int64) {
	key := fmt.Sprintf("otp_attempts:%d", telegramID)
	count, err := s.redis.Incr(ctx, key).Result()
	if err != nil {
		return
	}
	
	if count == 1 {
		s.redis.Expire(ctx, key, 15*time.Minute)
	}

	if count >= int64(maxOTPAttempts) {
		lockKey := fmt.Sprintf("otp_lockout:%d", telegramID)
		s.redis.Set(ctx, lockKey, "locked", otpLockDuration)
		s.redis.Del(ctx, key)
		logger.Warn("OTP brute-force lockout triggered (Redis)",
			zap.Int64("user_id", telegramID),
			zap.Int64("attempts", count),
		)
	}
}

func (s *AuthService) clearOTPLock(ctx context.Context, telegramID int64) {
	s.redis.Del(ctx, fmt.Sprintf("otp_lockout:%d", telegramID))
	s.redis.Del(ctx, fmt.Sprintf("otp_attempts:%d", telegramID))
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
