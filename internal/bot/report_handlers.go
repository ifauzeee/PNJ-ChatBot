package bot

import (
	"context"
	"fmt"
	"html"

	"github.com/pnj-anonymous-bot/internal/logger"
	"github.com/pnj-anonymous-bot/internal/metrics"
	"github.com/pnj-anonymous-bot/internal/models"
	"github.com/pnj-anonymous-bot/internal/validation"
	"go.uber.org/zap"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleReport(ctx context.Context, msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	partnerID, err := b.chat.GetPartner(ctx, telegramID)
	if err != nil || partnerID == 0 {
		b.sendMessage(telegramID, "⚠️ Kamu hanya bisa melaporkan partner saat sedang chat.", nil)
		return
	}

	logIfErr("set_state_awaiting_report", b.db.SetUserState(ctx, telegramID, models.StateAwaitingReport, fmt.Sprintf("%d", partnerID)))
	b.sendMessage(telegramID, `⚠️ *Laporkan Partner*

Tuliskan alasan kamu melaporkan partner ini.
Ketik /cancel untuk membatalkan.

📝 Alasan:`, nil)
}

func (b *Bot) handleReportInput(ctx context.Context, msg *tgbotapi.Message) {
	telegramID := msg.From.ID
	_, stateData, _ := b.db.GetUserState(ctx, telegramID)

	var reportedID int64
	_, _ = fmt.Sscanf(stateData, "%d", &reportedID)

	if reportedID == 0 {
		b.sendMessage(telegramID, "⚠️ Terjadi kesalahan. Coba lagi.", nil)
		logIfErr("set_state_none_report_error", b.db.SetUserState(ctx, telegramID, models.StateNone, ""))
		return
	}

	session, _ := b.db.GetActiveSession(ctx, telegramID)
	sessionID := int64(0)
	evidenceText := ""
	if session != nil {
		sessionID = session.ID
		evidenceText, _ = b.evidence.GetEvidence(ctx, sessionID)
	}

	reasonText := validation.SanitizeText(msg.Text)
	if errMsg := validation.ValidateText(reasonText, validation.ReportLimits); errMsg != "" {
		b.sendMessage(telegramID, errMsg, nil)
		return
	}

	newCount, err := b.profile.ReportUser(ctx, telegramID, reportedID, reasonText, evidenceText, sessionID)
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
		metrics.AutoBansTotal.Inc()
		b.sendMessageHTML(reportedID, "🚫 <b>Akun kamu telah diblokir otomatis oleh sistem karena telah mencapai batas laporan (3/3).</b> Kamu tidak bisa lagi menggunakan bot ini.", nil)
	}

	logIfErr("set_state_none_after_report", b.db.SetUserState(ctx, telegramID, models.StateNone, ""))
	metrics.ReportsTotal.Inc()
	b.sendMessageHTML(telegramID, "✅ <b>Laporan Terkirim!</b>\n\nTerima kasih atas laporanmu. Tim kami akan meninjau laporan ini.", nil)
}

func (b *Bot) handleBlock(ctx context.Context, msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	partnerID, err := b.chat.GetPartner(ctx, telegramID)
	if err != nil || partnerID == 0 {
		b.sendMessage(telegramID, "⚠️ Kamu hanya bisa memblock partner saat sedang chat.", nil)
		return
	}

	logIfErr("block_user", b.profile.BlockUser(ctx, telegramID, partnerID))

	if _, err := b.chat.StopChat(ctx, telegramID); err != nil {
		logger.Warn("Failed to stop chat after block", zap.Int64("user_id", telegramID), zap.Error(err))
	}

	metrics.BlocksTotal.Inc()
	b.sendMessage(partnerID, "👋 *Partner kamu telah memutus chat.*", nil)
	b.sendMessage(telegramID, "🚫 *Partner telah di-block.*\n\nKamu tidak akan dipasangkan dengan user ini lagi.", nil)
}
