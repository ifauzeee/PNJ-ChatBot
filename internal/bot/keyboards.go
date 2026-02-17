package bot

import (
	"fmt"
	"strings"

	"github.com/pnj-anonymous-bot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func GenderKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👨 Laki-laki", "gender:Laki-laki"),
			tgbotapi.NewInlineKeyboardButtonData("👩 Perempuan", "gender:Perempuan"),
		),
	)
}

func DepartmentKeyboard() tgbotapi.InlineKeyboardMarkup {
	depts := models.AllDepartments()
	var rows [][]tgbotapi.InlineKeyboardButton

	for _, dept := range depts {
		emoji := models.DepartmentEmoji(dept)
		btn := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%s %s", emoji, string(dept)),
			fmt.Sprintf("dept:%s", string(dept)),
		)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func SearchKeyboard() tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🎲 Acak (Semua Jurusan)", "search:any"),
	))

	depts := models.AllDepartments()
	for _, dept := range depts {
		emoji := models.DepartmentEmoji(dept)
		btn := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%s %s", emoji, string(dept)),
			fmt.Sprintf("search:%s", string(dept)),
		)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func ChatActionKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏭️ Next", "chat:next"),
			tgbotapi.NewInlineKeyboardButtonData("🛑 Stop", "chat:stop"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚠️ Report", "chat:report"),
			tgbotapi.NewInlineKeyboardButtonData("🚫 Block", "chat:block"),
		),
	)
}

func ConfirmKeyboard(confirmData, cancelData string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Ya", confirmData),
			tgbotapi.NewInlineKeyboardButtonData("❌ Tidak", cancelData),
		),
	)
}

func MainMenuKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔍 Cari Partner", "menu:search"),
			tgbotapi.NewInlineKeyboardButtonData("💬 Confession", "menu:confess"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📢 Whisper", "menu:whisper"),
			tgbotapi.NewInlineKeyboardButtonData("📋 Confessions", "menu:confessions"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👤 Profil", "menu:profile"),
			tgbotapi.NewInlineKeyboardButtonData("📊 Statistik", "menu:stats"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✏️ Edit Profil", "menu:edit"),
			tgbotapi.NewInlineKeyboardButtonData("❓ Bantuan", "menu:help"),
		),
	)
}

func EditProfileKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👤 Ubah Gender", "edit:gender"),
			tgbotapi.NewInlineKeyboardButtonData("🏛️ Ubah Jurusan", "edit:department"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Kembali", "menu:main"),
		),
	)
}

func ConfessionReactionKeyboard(confessionID int64, counts map[string]int) tgbotapi.InlineKeyboardMarkup {
	reactions := []struct {
		emoji string
		label string
	}{
		{"❤️", "❤️"},
		{"😂", "😂"},
		{"😢", "😢"},
		{"😮", "😮"},
		{"🔥", "🔥"},
	}

	var buttons []tgbotapi.InlineKeyboardButton
	for _, r := range reactions {
		count := counts[r.emoji]
		label := r.label
		if count > 0 {
			label = fmt.Sprintf("%s %d", r.label, count)
		}
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(
			label,
			fmt.Sprintf("react:%d:%s", confessionID, r.emoji),
		))
	}

	return tgbotapi.NewInlineKeyboardMarkup(
		buttons,
	)
}

func WhisperDeptKeyboard() tgbotapi.InlineKeyboardMarkup {
	depts := models.AllDepartments()
	var rows [][]tgbotapi.InlineKeyboardButton

	for _, dept := range depts {
		emoji := models.DepartmentEmoji(dept)
		btn := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%s %s", emoji, string(dept)),
			fmt.Sprintf("whisper:%s", string(dept)),
		)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Kembali", "menu:main"),
	))

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func CancelSearchKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Batalkan Pencarian", "search:cancel"),
		),
	)
}

func BackToMenuKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Menu Utama", "menu:main"),
		),
	)
}

func formatDepartmentList() string {
	depts := models.AllDepartments()
	var sb strings.Builder
	for i, dept := range depts {
		emoji := models.DepartmentEmoji(dept)
		sb.WriteString(fmt.Sprintf("  %d. %s %s\n", i+1, emoji, string(dept)))
	}
	return sb.String()
}
