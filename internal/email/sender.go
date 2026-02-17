package email

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/smtp"
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
		code[i] = byte('0' + num.Int64())
	}
	return string(code)
}

func (s *Sender) SendOTP(to, code string) error {
	subject := "🔐 Kode Verifikasi PNJ Anonymous Bot"

	body := fmt.Sprintf(`
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

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		s.cfg.SMTPFrom, to, subject, body)

	auth := smtp.PlainAuth("", s.cfg.SMTPUsername, s.cfg.SMTPPassword, s.cfg.SMTPHost)

	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)
	err := smtp.SendMail(addr, auth, s.cfg.SMTPUsername, []string{to}, []byte(msg))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func IsValidPNJEmail(emailAddr string) bool {
	emailAddr = strings.ToLower(strings.TrimSpace(emailAddr))

	validDomains := []string{
		"@mhsw.pnj.ac.id",
		"@pnj.ac.id",
	}

	for _, domain := range validDomains {
		if strings.HasSuffix(emailAddr, domain) {
			return true
		}
	}
	return false
}
