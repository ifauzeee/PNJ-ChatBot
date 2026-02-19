package email

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/pnj-anonymous-bot/internal/config"
)

type Sender struct {
	cfg *config.Config
}

func NewSender(cfg *config.Config) *Sender {
	return &Sender{cfg: cfg}
}

func GenerateOTP(length int) string {
	code := make([]byte, length)
	for i := 0; i < length; i++ {
		num, _ := rand.Int(rand.Reader, big.NewInt(10))
		code[i] = byte('0' + int32(num.Int64()))
	}
	return string(code)
}

func (s *Sender) SendOTP(to, code string) error {
	subject := "🔐 Kode Verifikasi PNJ Anonymous Bot"

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; margin: 0; padding: 0; background-color: #0f0f23; }
        .container { max-width: 500px; margin: 0 auto; padding: 40px 20px; }
        .card { background: linear-gradient(135deg, #1a1a3e 0%%, #2d2d6b 100%%); border-radius: 20px; padding: 40px; text-align: center; box-shadow: 0 20px 60px rgba(0,0,0,0.5); }
        .logo { font-size: 48px; margin-bottom: 10px; }
        .title { color: #ffffff; font-size: 24px; font-weight: 700; margin: 10px 0; }
        .subtitle { color: #a0a0cc; font-size: 14px; margin-bottom: 30px; }
        .otp-box { background: linear-gradient(135deg, #6366f1, #8b5cf6); border-radius: 16px; padding: 20px; margin: 20px 0; }
        .otp-code { font-size: 36px; font-weight: 800; color: #ffffff; letter-spacing: 12px; font-family: 'Courier New', monospace; }
        .warning { color: #ff6b6b; font-size: 12px; margin-top: 20px; }
        .footer { color: #666688; font-size: 11px; margin-top: 30px; }
        .divider { height: 1px; background: linear-gradient(90deg, transparent, #4a4a8a, transparent); margin: 20px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="card">
            <div class="logo">🎭</div>
            <div class="title">PNJ Anonymous Bot</div>
            <div class="subtitle">Verifikasi Email Mahasiswa PNJ</div>
            <div class="divider"></div>
            <p style="color: #c0c0e0; font-size: 14px;">Kode verifikasi kamu:</p>
            <div class="otp-box">
                <div class="otp-code">%s</div>
            </div>
            <p style="color: #a0a0cc; font-size: 13px;">
                Masukkan kode ini di bot Telegram untuk<br>menyelesaikan verifikasi akunmu.
            </p>
            <div class="warning">
                ⚠️ Kode ini berlaku selama %d menit.<br>
                Jangan bagikan kode ini kepada siapapun!
            </div>
            <div class="divider"></div>
            <div class="footer">
                Politeknik Negeri Jakarta<br>
                Jl. Prof. DR. G.A. Siwabessy, Kampus UI Depok 16425<br>
                © 2026 PNJ Anonymous Bot
            </div>
        </div>
    </div>
</body>
</html>`, code, s.cfg.OTPExpiryMinutes)

	type brevoRecipient struct {
		Email string `json:"email"`
	}

	type brevoSender struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	type brevoRequest struct {
		Sender      brevoSender      `json:"sender"`
		To          []brevoRecipient `json:"to"`
		Subject     string           `json:"subject"`
		HTMLContent string           `json:"htmlContent"`
	}

	senderName := "PNJ Anonymous Bot"
	senderEmail := s.cfg.SMTPUsername
	if strings.Contains(s.cfg.SMTPFrom, "<") {
		parts := strings.Split(s.cfg.SMTPFrom, "<")
		senderName = strings.TrimSpace(parts[0])
		senderEmail = strings.Trim(parts[1], "> ")
	}

	payload := brevoRequest{
		Sender: brevoSender{
			Name:  senderName,
			Email: senderEmail,
		},
		To:          []brevoRecipient{{Email: to}},
		Subject:     subject,
		HTMLContent: htmlBody,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.brevo.com/v3/smtp/email", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", s.cfg.BrevoAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to Brevo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errorRes map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errorRes); err != nil {
			return fmt.Errorf("brevo api error (status %d): (failed to decode error body)", resp.StatusCode)
		}
		return fmt.Errorf("brevo api error (status %d): %v", resp.StatusCode, errorRes)
	}

	return nil
}

func IsValidPNJEmail(emailAddr string) bool {
	emailAddr = strings.ToLower(strings.TrimSpace(emailAddr))

	validDomains := []string{
		"@mhsw.pnj.ac.id",
		"@stu.pnj.ac.id",
	}

	for _, domain := range validDomains {
		if strings.HasSuffix(emailAddr, domain) {
			return true
		}
	}
	return false
}
