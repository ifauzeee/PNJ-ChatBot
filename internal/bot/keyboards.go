package bot

import (
	"fmt"

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

func YearKeyboard() tgbotapi.InlineKeyboardMarkup {
	years := []int{2020, 2021, 2022, 2023, 2024, 2025}
	var rows [][]tgbotapi.InlineKeyboardButton

	for i := 0; i < len(years); i += 3 {
		var row []tgbotapi.InlineKeyboardButton
		for j := 0; j < 3 && i+j < len(years); j++ {
			year := years[i+j]
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("🎓 %d", year),
				fmt.Sprintf("year:%d", year),
			))
		}
		rows = append(rows, row)
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
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
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎲 Acak", "search:any"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👫 Berdasarkan Gender", "search:by_gender"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏛️ Berdasarkan Jurusan", "search:by_dept"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎓 Berdasarkan Angkatan", "search:by_year"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Kembali", "menu:main"),
		),
	)
}

func SearchYearKeyboard() tgbotapi.InlineKeyboardMarkup {
	years := []int{2020, 2021, 2022, 2023, 2024, 2025}
	var rows [][]tgbotapi.InlineKeyboardButton

	for i := 0; i < len(years); i += 3 {
		var row []tgbotapi.InlineKeyboardButton
		for j := 0; j < 3 && i+j < len(years); j++ {
			year := years[i+j]
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("🎓 Angkatan %d", year),
				fmt.Sprintf("search:year:%d", year),
			))
		}
		rows = append(rows, row)
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Kembali", "menu:search"),
	))

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func SearchDepartmentKeyboard() tgbotapi.InlineKeyboardMarkup {
	depts := models.AllDepartments()
	var rows [][]tgbotapi.InlineKeyboardButton

	for _, dept := range depts {
		emoji := models.DepartmentEmoji(dept)
		btn := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%s %s", emoji, string(dept)),
			fmt.Sprintf("search:dept:%s", string(dept)),
		)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Kembali", "menu:search"),
	))

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func SearchGenderKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👨 Cari Laki-laki", "search:gender:Laki-laki"),
			tgbotapi.NewInlineKeyboardButtonData("👩 Cari Perempuan", "search:gender:Perempuan"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Kembali", "menu:search"),
		),
	)
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
			tgbotapi.NewInlineKeyboardButtonData("🗳️ Polling", "menu:polls"),
			tgbotapi.NewInlineKeyboardButtonData("✏️ Edit Profil", "menu:edit"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚖️ Legal & About", "menu:about"),
			tgbotapi.NewInlineKeyboardButtonData("❓ Bantuan", "menu:help"),
		),
	)
}

func EditProfileKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👤 Ubah Gender", "edit:gender"),
			tgbotapi.NewInlineKeyboardButtonData("🎓 Ubah Angkatan", "edit:year"),
		),
		tgbotapi.NewInlineKeyboardRow(
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

func PollVoteKeyboard(pollID int64, options []*models.PollOption) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, opt := range options {
		label := opt.OptionText
		if opt.VoteCount > 0 {
			label = fmt.Sprintf("%s (%d)", opt.OptionText, opt.VoteCount)
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("vote:%d:%d", pollID, opt.ID)),
		))
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}
