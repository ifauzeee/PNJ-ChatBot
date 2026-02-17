package bot

import (
	"fmt"
	"html"

	"github.com/pnj-anonymous-bot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleReport(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	partnerID, err := b.chat.GetPartner(telegramID)
	if err != nil || partnerID == 0 {
		b.sendMessage(telegramID, "⚠️ Kamu hanya bisa melaporkan partner saat sedang chat.", nil)
		return
	}

	b.db.SetUserState(telegramID, models.StateAwaitingReport, fmt.Sprintf("%d", partnerID))
	b.sendMessage(telegramID, `⚠️ *Laporkan Partner*

Tuliskan alasan kamu melaporkan partner ini.
Ketik /cancel untuk membatalkan.

📝 Alasan:`, nil)
}

func (b *Bot) handleReportInput(msg *tgbotapi.Message) {
	telegramID := msg.From.ID
	_, stateData, _ := b.db.GetUserState(telegramID)

	var reportedID int64
	fmt.Sscanf(stateData, "%d", &reportedID)

	if reportedID == 0 {
		b.sendMessage(telegramID, "⚠️ Terjadi kesalahan. Coba lagi.", nil)
		b.db.SetUserState(telegramID, models.StateNone, "")
		return
	}

	session, _ := b.db.GetActiveSession(telegramID)
	sessionID := int64(0)
	if session != nil {
		sessionID = session.ID
	}

	newCount, err := b.profile.ReportUser(telegramID, reportedID, msg.Text, sessionID)
	if err != nil {
		b.sendMessageHTML(telegramID, fmt.Sprintf("⚠️ <b>%s</b>", html.EscapeString(err.Error())), nil)
		return
	}

	if newCount > 0 && newCount < b.cfg.AutoBanReportCount {
		warningMsg := fmt.Sprintf(`⚠️ <b>PERINGATAN MODERASI</b>

Akun kamu baru saja dilaporkan karena perilaku atau pesan yang tidak pantas.

📊 Status Laporan: <b>%d/%d</b>

Mohon patuhi aturan komunitas agar akun kamu tidak diblokir secara otomatis oleh sistem.`, newCount, b.cfg.AutoBanReportCount)

		b.sendMessageHTML(reportedID, warningMsg, nil)
	} else if newCount >= b.cfg.AutoBanReportCount {
		b.sendMessageHTML(reportedID, "🚫 <b>Akun kamu telah diblokir otomatis oleh sistem karena telah mencapai batas laporan (3/3).</b> Kamu tidak bisa lagi menggunakan bot ini.", nil)
	}

	b.db.SetUserState(telegramID, models.StateNone, "")
	b.sendMessageHTML(telegramID, "✅ <b>Laporan Terkirim!</b>\n\nTerima kasih atas laporanmu. Tim kami akan meninjau laporan ini.", nil)
}

func (b *Bot) handleBlock(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	partnerID, err := b.chat.GetPartner(telegramID)
	if err != nil || partnerID == 0 {
		b.sendMessage(telegramID, "⚠️ Kamu hanya bisa memblock partner saat sedang chat.", nil)
		return
	}

	b.profile.BlockUser(telegramID, partnerID)

	b.chat.StopChat(telegramID)

	b.sendMessage(partnerID, "👋 *Partner kamu telah memutus chat.*", nil)
	b.sendMessage(telegramID, "🚫 *Partner telah di-block.*\n\nKamu tidak akan dipasangkan dengan user ini lagi.", nil)
}
