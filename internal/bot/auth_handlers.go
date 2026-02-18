package bot

import (
	"fmt"
	"strings"

	"github.com/pnj-anonymous-bot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleRegist(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	user, err := b.auth.RegisterUser(telegramID)
	if err != nil {
		b.sendMessage(telegramID, "❌ Terjadi kesalahan. Coba lagi nanti.", nil)
		return
	}

	if user.IsVerified {
		b.sendMessage(telegramID, "✅ Kamu sudah terverifikasi!", nil)
		return
	}

	aboutText := `<b>⚖️ Informasi Hukum & Disclaimer</b>

━━━━━━━━━━━━━━━━━━━
<b>Disclaimer Afiliasi:</b>
Platform ini adalah layanan independen yang dikembangkan oleh sekelompok mahasiswa untuk tujuan sosial dan komunikasi antar mahasiswa. PNJ Anonymous Bot <b>TIDAK MEMILIKI HUBUNGAN AFILIASI</b> dengan manajemen Politeknik Negeri Jakarta (PNJ). Segala bentuk logo atau nama "PNJ" digunakan semata-mata untuk menunjukkan target demografis pengguna (mahasiswa PNJ).

<b>Tanggung Jawab Konten:</b>
Seluruh pesan, confession, dan polling yang dikirimkan melalui bot ini adalah tanggung jawab sepenuhnya dari <b>PENGIRIM (USER)</b>. Pengembang bot tidak bertanggung jawab atas segala bentuk kerugian, pencemaran nama baik, atau masalah hukum yang timbul akibat penyalahgunaan layanan ini.

<b>Privasi & Data:</b>
Kami hanya menyimpan alamat email PNJ untuk memastikan sistem hanya digunakan oleh mahasiswa aktif. Kami berkomitmen untuk menjaga kerahasiaan identitas anonim Anda dan tidak akan pernah membocorkan identitas pengirim pesan kecuali diminta secara resmi oleh pihak berwenang melalui jalur hukum yang berlaku di Indonesia.

<b>Persetujuan:</b>
Dengan menggunakan bot ini, Anda dianggap telah membaca dan menyetujui seluruh ketentuan di atas.

<i>Stay Anonymous, Stay Responsible.</i>`

	kb := LegalAgreementKeyboard()
	b.sendMessageHTML(telegramID, aboutText, &kb)
}

func (b *Bot) startEmailVerif(telegramID int64) {
	registText := `🔐 <b>Verifikasi Email</b>
━━━━━━━━━━━━━━━━━━━
Untuk menggunakan bot ini, kamu perlu verifikasi email PNJ kamu.

📧 <b>Ketik email PNJ kamu:</b>
Contoh: <i>nama@mhsw.pnj.ac.id / nama@stu.pnj.ac.id</i>

⚠️ Pastikan email kamu benar dan aktif.`

	_ = b.db.SetUserState(telegramID, models.StateAwaitingEmail, "")
	b.sendMessageHTML(telegramID, registText, nil)
}

func (b *Bot) handleEmailInput(msg *tgbotapi.Message) {
	telegramID := msg.From.ID
	emailAddr := strings.TrimSpace(msg.Text)

	err := b.auth.InitiateVerification(telegramID, emailAddr)
	if err != nil {
		b.sendMessage(telegramID, fmt.Sprintf("⚠️ %s\n\n📧 Silakan ketik email PNJ kamu:", err.Error()), nil)
		return
	}

	b.sendMessageHTML(telegramID, fmt.Sprintf(`📧 <b>Kode OTP Telah Dikirim!</b>

━━━━━━━━━━━━━━━━━━━
Email: <b>%s</b>
⏱️ Kode berlaku: <b>%d menit</b>
━━━━━━━━━━━━━━━━━━━

📬 Cek inbox email kamu dan masukkan kode 6 digit yang diterima.

🔢 <i>Ketik kode OTP kamu:</i>`,
		maskEmail(emailAddr), b.cfg.OTPExpiryMinutes), nil)
}

func (b *Bot) handleOTPInput(msg *tgbotapi.Message) {
	telegramID := msg.From.ID
	code := strings.TrimSpace(msg.Text)

	valid, err := b.auth.VerifyOTP(telegramID, code)
	if err != nil {
		b.sendMessage(telegramID, "❌ Terjadi kesalahan. Coba lagi.", nil)
		return
	}

	if !valid {
		b.sendMessage(telegramID, "❌ *Kode OTP salah atau sudah kedaluwarsa.*\n\n🔄 Gunakan /regist untuk mengirim ulang kode.", nil)
		return
	}

	b.sendMessage(telegramID, `✅ *Email Berhasil Diverifikasi!*

🎉 Selamat! Akun kamu telah terverifikasi.

Selanjutnya, isi profil kamu:`, nil)

	kb := GenderKeyboard()
	b.sendMessage(telegramID, "👤 *Pilih Gender Kamu:*", &kb)
}
