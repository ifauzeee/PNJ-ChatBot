package bot

import (
	"context"
	"strings"

	"github.com/pnj-anonymous-bot/internal/logger"
	"github.com/pnj-anonymous-bot/internal/metrics"
	"go.uber.org/zap"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func logIfErr(operation string, err error) {
	if err != nil {
		logger.Warn("Non-critical operation failed",
			zap.String("operation", operation),
			zap.Error(err),
		)
		metrics.DatabaseErrors.WithLabelValues(operation).Inc()
	}
}

func (b *Bot) sendAPI(operation string, c tgbotapi.Chattable) {
	if _, err := b.api.Send(c); err != nil {
		logger.Warn("Failed to send Telegram API message",
			zap.String("operation", operation),
			zap.Error(err),
		)
		metrics.TelegramAPIErrors.WithLabelValues(operation).Inc()
	}
}

func (b *Bot) deleteMessage(chatID int64, messageID int, operation string) {
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	b.sendAPI(operation, deleteMsg)
}

func maskEmail(emailAddr string) string {
	parts := strings.Split(emailAddr, "@")
	if len(parts) != 2 {
		return "***"
	}

	name := parts[0]
	if len(name) <= 3 {
		return name[:1] + "***@" + parts[1]
	}
	return name[:3] + "***@" + parts[1]
}

func escapeMarkdown(text string) string {
	replacer := strings.NewReplacer(
		"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]", "(", "\\(",
		")", "\\)", "~", "\\~", "`", "\\`", ">", "\\>", "#", "\\#",
		"+", "\\+", "-", "\\-", "=", "\\=", "|", "\\|", "{", "\\{",
		"}", "\\}", ".", "\\.", "!", "\\!",
	)
	return replacer.Replace(text)
}

func (b *Bot) forwardMedia(targetID int64, msg *tgbotapi.Message, captionPrefix string) {
	if msg.Sticker != nil {
		stickerCfg := tgbotapi.StickerConfig{
			BaseFile: tgbotapi.BaseFile{
				BaseChat: tgbotapi.BaseChat{ChatID: targetID},
				File:     tgbotapi.FileID(msg.Sticker.FileID),
			},
		}
		b.sendAPI("forward_sticker", stickerCfg)
	} else if msg.Photo != nil {
		photos := msg.Photo
		photo := photos[len(photos)-1]
		photoMsg := tgbotapi.NewPhoto(targetID, tgbotapi.FileID(photo.FileID))
		photoMsg.Caption = captionPrefix
		if msg.Caption != "" {
			photoMsg.Caption += "\n\n" + msg.Caption
		}
		b.sendAPI("forward_photo", photoMsg)
	} else if msg.Voice != nil {
		voice := tgbotapi.NewVoice(targetID, tgbotapi.FileID(msg.Voice.FileID))
		voice.Caption = captionPrefix
		b.sendAPI("forward_voice", voice)
	} else if msg.Video != nil {
		video := tgbotapi.NewVideo(targetID, tgbotapi.FileID(msg.Video.FileID))
		video.Caption = captionPrefix
		if msg.Caption != "" {
			video.Caption += "\n\n" + msg.Caption
		}
		b.sendAPI("forward_video", video)
	} else if msg.Document != nil {
		doc := tgbotapi.NewDocument(targetID, tgbotapi.FileID(msg.Document.FileID))
		doc.Caption = captionPrefix
		if msg.Caption != "" {
			doc.Caption += "\n\n" + msg.Caption
		}
		b.sendAPI("forward_document", doc)
	} else if msg.Animation != nil {
		anim := tgbotapi.NewAnimation(targetID, tgbotapi.FileID(msg.Animation.FileID))
		b.sendAPI("forward_animation", anim)
	}
}

func (b *Bot) isSafeMedia(ctx context.Context, msg *tgbotapi.Message) (bool, string) {
	if !b.moderation.IsEnabled() {
		return true, ""
	}

	var fileID string
	if len(msg.Photo) > 0 {
		photos := msg.Photo
		fileID = photos[len(photos)-1].FileID
	} else if msg.Sticker != nil {
		fileID = msg.Sticker.FileID
	} else if msg.Animation != nil {
		fileID = msg.Animation.FileID
	}

	if fileID == "" {
		return true, ""
	}

	url, err := b.api.GetFileDirectURL(fileID)
	if err != nil {
		logger.Warn("Failed to get file direct URL for moderation",
			zap.Error(err),
		)
		return true, ""
	}

	safe, reason, err := b.moderation.IsSafe(ctx, url)
	if err != nil {
		logger.Warn("Moderation check failed",
			zap.Error(err),
		)
	}
	if !safe {
		metrics.ModerationBlocked.Inc()
	}
	return safe, reason
}
