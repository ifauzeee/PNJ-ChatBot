package service

import (
	"testing"

	"github.com/pnj-anonymous-bot/internal/config"
)

type MockEmailSender struct {
	LastTo   string
	LastCode string
	Err      error
}

func (m *MockEmailSender) SendOTP(to, code string) error {
	m.LastTo = to
	m.LastCode = code
	return m.Err
}

func TestAuthServiceVerificationFlow(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.Config{
		OTPLength:        6,
		OTPExpiryMinutes: 10,
	}
	mockEmail := &MockEmailSender{}
	authSvc := NewAuthService(db, mockEmail, cfg)

	userID := int64(9001)
	emailAddr := "test@mhsw.pnj.ac.id"

	user, err := authSvc.RegisterUser(userID)
	if err != nil {
		t.Fatalf("RegisterUser failed: %v", err)
	}
	if user.TelegramID != userID {
		t.Errorf("Expected telegram ID %d, got %d", userID, user.TelegramID)
	}

	err = authSvc.InitiateVerification(userID, emailAddr)
	if err != nil {
		t.Fatalf("InitiateVerification failed: %v", err)
	}
	if mockEmail.LastTo != emailAddr {
		t.Errorf("Expected email sent to %s, got %s", emailAddr, mockEmail.LastTo)
	}
	if len(mockEmail.LastCode) != 6 {
		t.Errorf("Expected 6-digit OTP, got %s", mockEmail.LastCode)
	}

	verified, err := authSvc.VerifyOTP(userID, mockEmail.LastCode)
	if err != nil {
		t.Fatalf("VerifyOTP failed: %v", err)
	}
	if !verified {
		t.Errorf("Expected OTP to be verified")
	}

	isVerified, _ := authSvc.IsVerified(userID)
	if !isVerified {
		t.Errorf("User should be verified now")
	}
}

func TestAuthServiceInvalidOTP(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.Config{OTPLength: 6, OTPExpiryMinutes: 10}
	authSvc := NewAuthService(db, &MockEmailSender{}, cfg)

	userID := int64(9002)
	_, _ = authSvc.RegisterUser(userID)
	_ = authSvc.InitiateVerification(userID, "wrong@mhsw.pnj.ac.id")

	verified, err := authSvc.VerifyOTP(userID, "000000")
	if err != nil {
		t.Fatalf("VerifyOTP failed: %v", err)
	}
	if verified {
		t.Errorf("Should not verify wrong OTP")
	}
}

func TestAuthServiceRejectsInvalidDomain(t *testing.T) {
	db := setupTestDB(t)
	authSvc := NewAuthService(db, &MockEmailSender{}, &config.Config{})

	err := authSvc.InitiateVerification(9003, "test@gmail.com")
	if err == nil {
		t.Errorf("Expected error for invalid email domain")
	}
}
