package bot

import (
	"fmt"
	"strings"

	"github.com/pnj-anonymous-bot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleStart(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	user, err := b.auth.RegisterUser(telegramID)
	if err != nil {
		b.sendMessage(telegramID, "❌ Terjadi kesalahan. Coba lagi nanti.", nil)
		return
	}

	if !user.IsVerified {
		welcomeText := fmt.Sprintf(`🎭 *Selamat Datang di PNJ Anonymous Bot!*

Hai %s! 👋

Bot ini adalah platform anonim khusus untuk *mahasiswa Politeknik Negeri Jakarta* 🏛️

⚠️ *Email belum diverifikasi!*

Ketik /regist dan ikuti proses verifikasi email PNJ kamu.`, msg.From.FirstName)

		b.sendMessage(telegramID, welcomeText, nil)
		return
	}

	if string(user.Gender) == "" {
		b.sendMessage(telegramID, "👤 *Pilih Gender Kamu:*", &tgbotapi.InlineKeyboardMarkup{})
		kb := GenderKeyboard()
		b.sendMessage(telegramID, "👇 Silakan pilih:", &kb)
		b.db.SetUserState(telegramID, models.StateAwaitingGender, "")
		return
	}

	if string(user.Department) == "" {
		kb := DepartmentKeyboard()
		b.sendMessage(telegramID, "🏛️ *Pilih Jurusan Kamu:*\n\nPilih jurusan di bawah ini:", &kb)
		b.db.SetUserState(telegramID, models.StateAwaitingDept, "")
		return
	}

	b.showMainMenu(telegramID, user)
	b.showMainMenu(telegramID, user)
}

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

	registText := `🔐 *Verifikasi Email*
━━━━━━━━━━━━━━━━━━━
Untuk menggunakan bot ini, kamu perlu verifikasi email PNJ kamu.

📧 *Ketik email PNJ kamu:*
Contoh: _nama@mhsw.pnj.ac.id_

Domain yang diterima:
• @mhsw.pnj.ac.id (Mahasiswa)
• @stu.pnj.ac.id (Dosen/Staff)`

	b.db.SetUserState(telegramID, models.StateAwaitingEmail, "")
	b.sendMessage(telegramID, registText, nil)
}

func (b *Bot) showMainMenu(telegramID int64, user *models.User) {
	onlineCount, _ := b.db.GetOnlineUserCount()
	queueCount, _ := b.chat.GetQueueCount()

	menuText := fmt.Sprintf(`🎭 *PNJ Anonymous Bot*

━━━━━━━━━━━━━━━━━━━
👤 *Profil Kamu*
━━━━━━━━━━━━━━━━━━━
%s %s | %s %s
🏷️ %s
📧 ||%s||

━━━━━━━━━━━━━━━━━━━
📊 *Info Bot*
━━━━━━━━━━━━━━━━━━━
👥 Pengguna terdaftar: *%d*
🔍 Sedang mencari: *%d*

━━━━━━━━━━━━━━━━━━━
✨ Pilih menu di bawah ini:`,
		models.GenderEmoji(user.Gender), string(user.Gender),
		models.DepartmentEmoji(user.Department), string(user.Department),
		user.DisplayName,
		maskEmail(user.Email),
		onlineCount,
		queueCount,
	)

	kb := MainMenuKeyboard()
	b.sendMessage(telegramID, menuText, &kb)
}

func (b *Bot) handleHelp(msg *tgbotapi.Message) {
	helpText := `❓ *Panduan PNJ Anonymous Bot*

━━━━━━━━━━━━━━━━━━━
🔍 *Chat Anonim*
━━━━━━━━━━━━━━━━━━━
/search — Cari partner chat
/next — Skip ke partner baru
/stop — Hentikan chat

━━━━━━━━━━━━━━━━━━━
💬 *Confession & Whisper*
━━━━━━━━━━━━━━━━━━━
/confess — Kirim confession anonim
/confessions — Lihat confession terbaru
/whisper — Kirim pesan ke jurusan

━━━━━━━━━━━━━━━━━━━
👤 *Profil & Settings*
━━━━━━━━━━━━━━━━━━━
/profile — Lihat profil kamu
/stats — Statistik interaksi
/edit — Edit profil

━━━━━━━━━━━━━━━━━━━
🛡️ *Keamanan*
━━━━━━━━━━━━━━━━━━━
/report — Laporkan partner
/block — Block partner
/cancel — Batalkan aksi saat ini

━━━━━━━━━━━━━━━━━━━
📌 *Aturan:*
1️⃣ Jaga kesopanan dalam berkomunikasi
2️⃣ Dilarang menyebarkan konten SARA
3️⃣ Dilarang spam atau flood
4️⃣ Hormati privasi pengguna lain
5️⃣ Pelanggaran = auto-ban setelah 3 report

_Politeknik Negeri Jakarta © 2026_`

	kb := BackToMenuKeyboard()
	b.sendMessage(msg.From.ID, helpText, &kb)
}

func (b *Bot) handleCancel(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	state, _, _ := b.db.GetUserState(telegramID)

	switch state {
	case models.StateSearching:
		b.chat.CancelSearch(telegramID)
		b.sendMessage(telegramID, "❌ Pencarian dibatalkan.", nil)
	case models.StateAwaitingConfess:
		b.db.SetUserState(telegramID, models.StateNone, "")
		b.sendMessage(telegramID, "❌ Confession dibatalkan.", nil)
	case models.StateAwaitingWhisper, models.StateAwaitingWhisperDept:
		b.db.SetUserState(telegramID, models.StateNone, "")
		b.sendMessage(telegramID, "❌ Whisper dibatalkan.", nil)
	case models.StateAwaitingReport:
		b.db.SetUserState(telegramID, models.StateNone, "")
		b.sendMessage(telegramID, "❌ Report dibatalkan.", nil)
	default:
		b.sendMessage(telegramID, "💡 Tidak ada aksi yang perlu dibatalkan.", nil)
	}
}

func (b *Bot) handleSearch(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	state, _, _ := b.db.GetUserState(telegramID)
	if state == models.StateInChat {
		b.sendMessage(telegramID, "⚠️ Kamu masih dalam sesi chat!\nGunakan /stop untuk menghentikan atau /next untuk partner baru.", nil)
		return
	}

	if state == models.StateSearching {
		b.sendMessage(telegramID, "⏳ Kamu sudah dalam antrian pencarian. Tunggu sebentar ya!", nil)
		return
	}

	args := msg.CommandArguments()
	if args != "" {
		b.startSearch(telegramID, args)
		return
	}

	kb := SearchKeyboard()
	b.sendMessage(telegramID, "🔍 *Cari Partner Chat Anonim*\n\nPilih filter pencarian:", &kb)
}

func (b *Bot) startSearch(telegramID int64, preferredDept string) {
	if preferredDept == "any" {
		preferredDept = ""
	}

	matchID, err := b.chat.SearchPartner(telegramID, preferredDept)
	if err != nil {
		b.sendMessage(telegramID, fmt.Sprintf("⚠️ %s", err.Error()), nil)
		return
	}

	if matchID > 0 {

		b.notifyMatchFound(telegramID, matchID)
	} else {

		queueCount, _ := b.chat.GetQueueCount()
		kb := CancelSearchKeyboard()

		searchText := fmt.Sprintf(`🔍 *Mencari Partner...*

⏳ Kamu telah masuk ke antrian
👥 Orang dalam antrian: *%d*

_Menunggu partner yang cocok..._
_Kamu akan diberi notifikasi ketika partner ditemukan_`, queueCount)

		b.sendMessage(telegramID, searchText, &kb)
	}
}

func (b *Bot) notifyMatchFound(user1ID, user2ID int64) {

	gender2, dept2, _ := b.chat.GetPartnerInfo(user2ID)

	gender1, dept1, _ := b.chat.GetPartnerInfo(user1ID)

	kb := ChatActionKeyboard()

	msg1 := fmt.Sprintf(`🎉 *Partner Ditemukan!*

━━━━━━━━━━━━━━━━━━━
Partner kamu:
%s %s | %s %s
━━━━━━━━━━━━━━━━━━━

💬 Mulai ngobrol sekarang!
Semua pesan akan diteruskan secara *anonim*.

_Ketik pesan untuk memulai..._`,
		models.GenderEmoji(models.Gender(gender2)), gender2,
		models.DepartmentEmoji(models.Department(dept2)), dept2)

	msg2 := fmt.Sprintf(`🎉 *Partner Ditemukan!*

━━━━━━━━━━━━━━━━━━━
Partner kamu:
%s %s | %s %s
━━━━━━━━━━━━━━━━━━━

💬 Mulai ngobrol sekarang!
Semua pesan akan diteruskan secara *anonim*.

_Ketik pesan untuk memulai..._`,
		models.GenderEmoji(models.Gender(gender1)), gender1,
		models.DepartmentEmoji(models.Department(dept1)), dept1)

	b.sendMessage(user1ID, msg1, &kb)
	b.sendMessage(user2ID, msg2, &kb)
}

func (b *Bot) handleNext(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	partnerID, err := b.chat.NextPartner(telegramID)
	if err != nil {
		b.sendMessage(telegramID, fmt.Sprintf("⚠️ %s", err.Error()), nil)
		return
	}

	if partnerID > 0 {

		b.sendMessage(partnerID, "👋 *Partner kamu telah memutus chat.*\n\nGunakan /search untuk mencari partner baru.", nil)
	}

	b.sendMessage(telegramID, "⏭️ *Mencari partner baru...*", nil)
	b.startSearch(telegramID, "")
}

func (b *Bot) handleStop(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	state, _, _ := b.db.GetUserState(telegramID)
	if state == models.StateSearching {
		b.chat.CancelSearch(telegramID)
		b.sendMessage(telegramID, "🛑 Pencarian dihentikan.", nil)
		return
	}

	partnerID, err := b.chat.StopChat(telegramID)
	if err != nil {
		b.sendMessage(telegramID, "⚠️ Tidak ada chat aktif saat ini.", nil)
		return
	}

	if partnerID > 0 {
		b.sendMessage(partnerID, "👋 *Partner kamu telah memutus chat.*\n\nGunakan /search untuk mencari partner baru.", nil)
		b.sendMessage(telegramID, "🛑 *Chat dihentikan.*\n\nGunakan /search untuk cari partner baru.", nil)
	} else {
		b.sendMessage(telegramID, "💡 Tidak ada chat aktif saat ini.", nil)
	}
}

func (b *Bot) handleChatMessage(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	partnerID, err := b.chat.GetPartner(telegramID)
	if err != nil || partnerID == 0 {
		b.db.SetUserState(telegramID, models.StateNone, "")
		b.sendMessage(telegramID, "⚠️ Chat tidak aktif. Gunakan /search untuk mencari partner.", nil)
		return
	}

	if msg.Text != "" {
		b.sendMessage(partnerID, fmt.Sprintf("💬 *Stranger:*\n%s", escapeMarkdown(msg.Text)), nil)
	} else if msg.Sticker != nil {
		stickerMsg := tgbotapi.NewMessage(partnerID, "")
		stickerCfg := tgbotapi.StickerConfig{
			BaseFile: tgbotapi.BaseFile{
				BaseChat: tgbotapi.BaseChat{ChatID: partnerID},
				File:     tgbotapi.FileID(msg.Sticker.FileID),
			},
		}
		b.api.Send(stickerCfg)
		_ = stickerMsg
	} else if msg.Photo != nil {
		photos := msg.Photo
		photo := photos[len(photos)-1]
		photoMsg := tgbotapi.NewPhoto(partnerID, tgbotapi.FileID(photo.FileID))
		if msg.Caption != "" {
			photoMsg.Caption = fmt.Sprintf("💬 Stranger: %s", msg.Caption)
		}
		b.api.Send(photoMsg)
	} else if msg.Voice != nil {
		voice := tgbotapi.NewVoice(partnerID, tgbotapi.FileID(msg.Voice.FileID))
		b.api.Send(voice)
	} else if msg.Video != nil {
		video := tgbotapi.NewVideo(partnerID, tgbotapi.FileID(msg.Video.FileID))
		if msg.Caption != "" {
			video.Caption = fmt.Sprintf("💬 Stranger: %s", msg.Caption)
		}
		b.api.Send(video)
	} else if msg.Document != nil {
		doc := tgbotapi.NewDocument(partnerID, tgbotapi.FileID(msg.Document.FileID))
		if msg.Caption != "" {
			doc.Caption = fmt.Sprintf("💬 Stranger: %s", msg.Caption)
		}
		b.api.Send(doc)
	} else if msg.Animation != nil {
		anim := tgbotapi.NewAnimation(partnerID, tgbotapi.FileID(msg.Animation.FileID))
		b.api.Send(anim)
	} else if msg.VideoNote != nil {
		vnCfg := tgbotapi.VideoNoteConfig{
			BaseFile: tgbotapi.BaseFile{
				BaseChat: tgbotapi.BaseChat{ChatID: partnerID},
				File:     tgbotapi.FileID(msg.VideoNote.FileID),
			},
			Length: msg.VideoNote.Length,
		}
		b.api.Send(vnCfg)
	} else {
		b.sendMessage(telegramID, "⚠️ Tipe pesan ini tidak didukung.", nil)
	}
}

func (b *Bot) handleConfess(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	b.db.SetUserState(telegramID, models.StateAwaitingConfess, "")

	b.sendMessage(telegramID, `💬 *Tulis Confession Kamu*

━━━━━━━━━━━━━━━━━━━
Kirim confession anonim yang bisa dibaca semua pengguna.

📝 Ketik confession kamu sekarang...
Atau ketik /cancel untuk membatalkan.

⚠️ _Confession akan menampilkan jurusan kamu tapi TIDAK identitas kamu._`, nil)
}

func (b *Bot) handleConfessionInput(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	if msg.Text == "" {
		b.sendMessage(telegramID, "⚠️ Confession harus berupa teks.", nil)
		return
	}

	if len(msg.Text) < 10 {
		b.sendMessage(telegramID, "⚠️ Confession terlalu pendek. Minimal 10 karakter.", nil)
		return
	}

	if len(msg.Text) > 1000 {
		b.sendMessage(telegramID, "⚠️ Confession terlalu panjang. Maksimal 1000 karakter.", nil)
		return
	}

	confession, err := b.confession.CreateConfession(telegramID, msg.Text)
	if err != nil {
		b.sendMessage(telegramID, fmt.Sprintf("⚠️ %s", err.Error()), nil)
		return
	}

	b.db.SetUserState(telegramID, models.StateNone, "")

	b.sendMessage(telegramID, fmt.Sprintf(`✅ *Confession Terkirim!*

📝 Confession #%d berhasil dikirim.
Confession kamu sekarang bisa dilihat semua pengguna melalui /confessions.`, confession.ID), nil)
}

func (b *Bot) handleConfessions(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	confessions, err := b.confession.GetLatestConfessions(10)
	if err != nil {
		b.sendMessage(telegramID, "❌ Gagal mengambil confession.", nil)
		return
	}

	if len(confessions) == 0 {
		b.sendMessage(telegramID, "📋 Belum ada confession. Jadilah yang pertama dengan /confess!", nil)
		return
	}

	header := "📋 *Confession Terbaru*\n━━━━━━━━━━━━━━━━━━━\n\n"

	for _, c := range confessions {
		emoji := models.DepartmentEmoji(models.Department(c.Department))
		counts, _ := b.confession.GetReactionCounts(c.ID)

		reactionStr := ""
		for r, count := range counts {
			reactionStr += fmt.Sprintf("%s%d ", r, count)
		}

		text := fmt.Sprintf(`💬 *#%d* | %s %s
%s
%s
─────────────────
`, c.ID, emoji, c.Department, escapeMarkdown(c.Content), reactionStr)

		header += text
	}

	header += "\n_React ke confession: ketik_ /react <id> <emoji>"

	b.sendMessage(telegramID, header, nil)
}

func (b *Bot) handleWhisper(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	kb := WhisperDeptKeyboard()
	b.sendMessage(telegramID, `📢 *Whisper - Pesan Anonim ke Jurusan*

━━━━━━━━━━━━━━━━━━━
Kirim pesan anonim ke semua mahasiswa di jurusan tertentu!

🎯 Pilih jurusan tujuan:`, &kb)
}

func (b *Bot) handleWhisperInput(msg *tgbotapi.Message, targetDept string) {
	telegramID := msg.From.ID

	if msg.Text == "" || len(msg.Text) < 5 {
		b.sendMessage(telegramID, "⚠️ Whisper terlalu pendek. Minimal 5 karakter.", nil)
		return
	}

	if len(msg.Text) > 500 {
		b.sendMessage(telegramID, "⚠️ Whisper terlalu panjang. Maksimal 500 karakter.", nil)
		return
	}

	targets, err := b.profile.SendWhisper(telegramID, targetDept, msg.Text)
	if err != nil {
		b.sendMessage(telegramID, fmt.Sprintf("⚠️ %s", err.Error()), nil)
		return
	}

	b.db.SetUserState(telegramID, models.StateNone, "")

	user, _ := b.db.GetUser(telegramID)
	senderDept := ""
	senderGender := ""
	if user != nil {
		senderDept = string(user.Department)
		senderGender = string(user.Gender)
	}

	for _, targetID := range targets {
		whisperMsg := fmt.Sprintf(`📢 *Whisper dari %s %s*

━━━━━━━━━━━━━━━━━━━
%s %s | %s
━━━━━━━━━━━━━━━━━━━
%s

_Pesan anonim untuk jurusan %s_`,
			models.DepartmentEmoji(models.Department(senderDept)), senderDept,
			models.GenderEmoji(models.Gender(senderGender)), senderGender,
			models.DepartmentEmoji(models.Department(senderDept)),
			escapeMarkdown(msg.Text),
			targetDept,
		)
		b.sendMessage(targetID, whisperMsg, nil)
	}

	b.sendMessage(telegramID, fmt.Sprintf("✅ *Whisper Terkirim!*\n\n📤 Dikirim ke *%d* mahasiswa %s %s",
		len(targets),
		models.DepartmentEmoji(models.Department(targetDept)),
		targetDept,
	), nil)
}

func (b *Bot) handleProfile(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	user, err := b.profile.GetProfile(telegramID)
	if err != nil || user == nil {
		b.sendMessage(telegramID, "❌ Gagal memuat profil.", nil)
		return
	}

	totalChats, totalConfessions, totalReactions, daysSince, _ := b.profile.GetStats(telegramID)

	profileText := fmt.Sprintf(`👤 *Profil Kamu*

━━━━━━━━━━━━━━━━━━━
🏷️ *Nama Anonim:* %s
%s *Gender:* %s
%s *Jurusan:* %s
📧 *Email:* ||%s||
━━━━━━━━━━━━━━━━━━━
📊 *Statistik:*
💬 Total Chat: *%d*
📝 Confessions: *%d*
❤️ Reactions Diterima: *%d*
📅 Hari Aktif: *%d*
━━━━━━━━━━━━━━━━━━━
⚠️ Report Count: %d/3`,
		user.DisplayName,
		models.GenderEmoji(user.Gender), string(user.Gender),
		models.DepartmentEmoji(user.Department), string(user.Department),
		maskEmail(user.Email),
		totalChats,
		totalConfessions,
		totalReactions,
		daysSince,
		user.ReportCount,
	)

	kb := BackToMenuKeyboard()
	b.sendMessage(telegramID, profileText, &kb)
}

func (b *Bot) handleStats(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	totalChats, totalConfessions, totalReactions, daysSince, err := b.profile.GetStats(telegramID)
	if err != nil {
		b.sendMessage(telegramID, "❌ Gagal memuat statistik.", nil)
		return
	}

	statsText := fmt.Sprintf(`📊 *Statistik Kamu*

━━━━━━━━━━━━━━━━━━━
💬 Total Chat: *%d*
📝 Confession Dibuat: *%d*
❤️ Reactions Diterima: *%d*
📅 Hari Sejak Bergabung: *%d*
━━━━━━━━━━━━━━━━━━━

_Terus berinteraksi untuk meningkatkan statistik kamu!_ 🚀`,
		totalChats, totalConfessions, totalReactions, daysSince)

	kb := BackToMenuKeyboard()
	b.sendMessage(telegramID, statsText, &kb)
}

func (b *Bot) handleEdit(msg *tgbotapi.Message) {
	kb := EditProfileKeyboard()
	b.sendMessage(msg.From.ID, "✏️ *Edit Profil*\n\nApa yang ingin kamu ubah?", &kb)
}

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

	err := b.profile.ReportUser(telegramID, reportedID, msg.Text, sessionID)
	if err != nil {
		b.sendMessage(telegramID, fmt.Sprintf("⚠️ %s", err.Error()), nil)
		return
	}

	b.db.SetUserState(telegramID, models.StateNone, "")
	b.sendMessage(telegramID, "✅ *Laporan Terkirim!*\n\nTerima kasih atas laporanmu. Tim kami akan meninjau laporan ini.", nil)
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

func (b *Bot) handleEmailInput(msg *tgbotapi.Message) {
	telegramID := msg.From.ID
	emailAddr := strings.TrimSpace(msg.Text)

	err := b.auth.InitiateVerification(telegramID, emailAddr)
	if err != nil {
		b.sendMessage(telegramID, fmt.Sprintf("⚠️ %s\n\n📧 Silakan ketik email PNJ kamu:", err.Error()), nil)
		return
	}

	b.sendMessage(telegramID, fmt.Sprintf(`📧 *Kode OTP Telah Dikirim!*

━━━━━━━━━━━━━━━━━━━
Email: *%s*
⏱️ Kode berlaku: *%d menit*
━━━━━━━━━━━━━━━━━━━

📬 Cek inbox email kamu dan masukkan kode 6 digit yang diterima.

🔢 _Ketik kode OTP kamu:_`,
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
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(text)
}
