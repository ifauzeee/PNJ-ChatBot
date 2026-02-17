package bot

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func maskEmail(emailAddr string) string {
	parts := strings.Split(emailAddr, "@")
	if len(parts) != 2 {
		return emailAddr
	}

	name := parts[0]
	if len(name) <= 3 {
		return name[:1] + "***@" + parts[1]
	}
	return name[:3] + "***@" + parts[1]
}

func escapeMarkdown(text string) string {
	replacer := strings.NewReplacer(
		"*", "\\*",
		"_", "\\_",
		"[", "\\[",
		"`", "\\`",
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
		b.api.Send(stickerCfg)
	} else if msg.Photo != nil {
		photos := msg.Photo
		photo := photos[len(photos)-1]
		photoMsg := tgbotapi.NewPhoto(targetID, tgbotapi.FileID(photo.FileID))
		photoMsg.Caption = captionPrefix
		if msg.Caption != "" {
			photoMsg.Caption += "\n\n" + msg.Caption
		}
		b.api.Send(photoMsg)
	} else if msg.Voice != nil {
		voice := tgbotapi.NewVoice(targetID, tgbotapi.FileID(msg.Voice.FileID))
		voice.Caption = captionPrefix
		b.api.Send(voice)
	} else if msg.Video != nil {
		video := tgbotapi.NewVideo(targetID, tgbotapi.FileID(msg.Video.FileID))
		video.Caption = captionPrefix
		if msg.Caption != "" {
			video.Caption += "\n\n" + msg.Caption
		}
		b.api.Send(video)
	} else if msg.Document != nil {
		doc := tgbotapi.NewDocument(targetID, tgbotapi.FileID(msg.Document.FileID))
		doc.Caption = captionPrefix
		if msg.Caption != "" {
			doc.Caption += "\n\n" + msg.Caption
		}
		b.api.Send(doc)
	} else if msg.Animation != nil {
		anim := tgbotapi.NewAnimation(targetID, tgbotapi.FileID(msg.Animation.FileID))
		b.api.Send(anim)
	}
}

func (b *Bot) isSafeMedia(msg *tgbotapi.Message) (bool, string) {
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
		return true, ""
	}

	safe, reason, _ := b.moderation.IsSafe(url)
	return safe, reason
}
