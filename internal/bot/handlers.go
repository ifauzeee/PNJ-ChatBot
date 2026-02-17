package bot

import (
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

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
		welcomeText := fmt.Sprintf(`<b>🎭 Selamat Datang di PNJ Anonymous Bot!</b>

Hai <b>%s</b>! 👋

Bot ini adalah platform anonim khusus untuk <b>mahasiswa Politeknik Negeri Jakarta</b> 🏛️

━━━━━━━━━━━━━━━━━━━
⚖️ <b>DISCLAIMER & LEGAL:</b>
Bot ini adalah <b>PROYEK INDEPENDEN</b> yang dibuat oleh mahasiswa untuk mahasiswa.
Bot ini <b>TIDAK berafiliasi, disponsori, atau disetujui secara resmi</b> oleh pihak institusi Politeknik Negeri Jakarta (PNJ).
Segala konten dan interaksi di dalam bot ini adalah tanggung jawab masing-masing pengguna.
━━━━━━━━━━━━━━━━━━━

⚠️ <b>Email belum diverifikasi!</b>

Ketik /regist dan ikuti proses verifikasi email PNJ kamu.`, html.EscapeString(msg.From.FirstName))

		b.sendMessageHTML(telegramID, welcomeText, nil)
		return
	}

	if string(user.Gender) == "" {
		b.sendMessage(telegramID, "👤 *Pilih Gender Kamu:*", nil)
		kb := GenderKeyboard()
		b.sendMessage(telegramID, "👇 Silakan pilih:", &kb)
		b.db.SetUserState(telegramID, models.StateAwaitingGender, "")
		return
	}

	if user.Year == 0 {
		kb := YearKeyboard()
		b.sendMessage(telegramID, "🎓 *Pilih Tahun Angkatan Kamu:*", &kb)
		b.db.SetUserState(telegramID, models.StateAwaitingYear, "")
		return
	}

	if string(user.Department) == "" {
		kb := DepartmentKeyboard()
		b.sendMessage(telegramID, "🏛️ *Pilih Jurusan Kamu:*\n\nPilih jurusan di bawah ini:", &kb)
		b.db.SetUserState(telegramID, models.StateAwaitingDept, "")
		return
	}

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

	b.db.SetUserState(telegramID, models.StateAwaitingEmail, "")
	b.sendMessageHTML(telegramID, registText, nil)
}

func (b *Bot) showMainMenu(telegramID int64, user *models.User) {
	if user == nil {
		user, _ = b.db.GetUser(telegramID)
	}
	if user == nil {
		return
	}

	onlineCount, _ := b.db.GetOnlineUserCount()
	queueCount, _ := b.chat.GetQueueCount()

	menuText := fmt.Sprintf(`🎭 <b>PNJ Anonymous Bot</b>

━━━━━━━━━━━━━━━━━━━
👤 <b>Profil Kamu</b>
━━━━━━━━━━━━━━━━━━━
%s %s | %s %s
🏷️ %s
📧 <tg-spoiler>%s</tg-spoiler>

━━━━━━━━━━━━━━━━━━━
📊 <b>Info Bot</b>
━━━━━━━━━━━━━━━━━━━
Pengguna online: <b>%d</b>
Antrian chat: <b>%d</b>

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
	b.sendMessageHTML(telegramID, menuText, &kb)
}

func (b *Bot) handleHelp(msg *tgbotapi.Message) {
	helpText := `<b>❓ Panduan PNJ Anonymous Bot</b>

━━━━━━━━━━━━━━━━━━━
🔍 <b>Chat Anonim</b>
/search — Cari partner chat
/next — Skip ke partner baru
/stop — Hentikan chat

💬 <b>Fitur Interaksi</b>
/confess — Kirim confession anonim
/reply — Balas confession
/poll — Buat polling anonim
/whisper — Pesan ke jurusan
/circles — Gabung circle (group chat)
/leave_circle — Keluar dari circle

👤 <b>Profil & Achievement</b>
/profile — Lihat profil & lencana
/stats — Statistik kamu
/edit — Edit data diri

🛡️ <b>Keamanan & Legal</b>
/report — Laporkan pelanggaran
/about — Informasi hukum & privasi
/cancel — Batalkan aksi

⚖️ <b>Ketentuan Layanan:</b>
1. Bot ini <b>UNOFFICIAL</b> (Bukan resmi dari PNJ).
2. Pengguna wajib menjaga etika & kesopanan.
3. Konten SARA/Pelecehan = <b>BANNED PERMANEN.</b>
4. Kami tidak menyimpan data pribadi selain email PNJ untuk verifikasi.

<i>Dibuat dengan ❤️ oleh Mahasiswa PNJ (Unofficial Project)</i>`

	kb := BackToMenuKeyboard()
	b.sendMessageHTML(msg.From.ID, helpText, &kb)
}

func (b *Bot) handleAbout(msg *tgbotapi.Message) {
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

	kb := BackToMenuKeyboard()
	b.sendMessageHTML(msg.From.ID, aboutText, &kb)
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
	case models.StateInCircle:
		b.handleLeaveCircle(msg)
	case models.StateAwaitingRoomName, models.StateAwaitingRoomDesc:
		b.db.SetUserState(telegramID, models.StateNone, "")
		b.sendMessage(telegramID, "❌ Pembuatan circle dibatalkan.", nil)
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
		parts := strings.Fields(args)
		preferredDept := ""
		preferredGender := ""
		preferredYear := 0

		for _, part := range parts {
			if models.IsValidDepartment(part) {
				preferredDept = part
			} else if part == "Laki-laki" || part == "Perempuan" {
				preferredGender = part
			} else if year, err := strconv.Atoi(part); err == nil && year >= 2018 && year <= 2026 {
				preferredYear = year
			}
		}
		b.startSearch(telegramID, preferredDept, preferredGender, preferredYear)
		return
	}

	kb := SearchKeyboard()
	b.sendMessage(telegramID, "🔍 *Cari Partner Chat Anonim*\n\nPilih filter pencarian:", &kb)
}

func (b *Bot) startSearch(telegramID int64, preferredDept, preferredGender string, preferredYear int) {
	if preferredDept == "any" {
		preferredDept = ""
	}

	b.room.LeaveRoom(telegramID)

	matchID, err := b.chat.SearchPartner(telegramID, preferredDept, preferredGender, preferredYear)
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
	gender1, dept1, year1, _ := b.chat.GetPartnerInfo(user1ID)
	gender2, dept2, year2, _ := b.chat.GetPartnerInfo(user2ID)

	kb := ChatActionKeyboard()

	msg1 := fmt.Sprintf(`<b>🎉 Partner Ditemukan!</b>

━━━━━━━━━━━━━━━━━━━
Partner kamu:
%s %s | 🎓 %d
%s %s
━━━━━━━━━━━━━━━━━━━

💬 Mulai ngobrol sekarang!
Semua pesan akan diteruskan secara <b>anonim</b>.

<i>Ketik pesan untuk memulai...</i>`,
		models.GenderEmoji(models.Gender(gender2)), html.EscapeString(gender2), year2,
		models.DepartmentEmoji(models.Department(dept2)), html.EscapeString(dept2))

	msg2 := fmt.Sprintf(`<b>🎉 Partner Ditemukan!</b>

━━━━━━━━━━━━━━━━━━━
Partner kamu:
%s %s | 🎓 %d
%s %s
━━━━━━━━━━━━━━━━━━━

💬 Mulai ngobrol sekarang!
Semua pesan akan diteruskan secara <b>anonim</b>.

<i>Ketik pesan untuk memulai...</i>`,
		models.GenderEmoji(models.Gender(gender1)), html.EscapeString(gender1), year1,
		models.DepartmentEmoji(models.Department(dept1)), html.EscapeString(dept1))

	b.sendMessageHTML(user1ID, msg1, &kb)
	b.sendMessageHTML(user2ID, msg2, &kb)
}

func (b *Bot) handleNext(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	state, _, _ := b.db.GetUserState(telegramID)

	if state == models.StateInCircle {
		room, _ := b.room.GetUserRoom(telegramID)
		roomName := "Circle"
		if room != nil {
			roomName = room.Name
		}

		kb := ConfirmKeyboard("circle:leave_next", "circle:stay")
		b.sendMessageHTML(telegramID, fmt.Sprintf(`⚠️ <b>Kamu sedang berada di %s</b>

Perintah /next hanya digunakan untuk mencari partner Private Chat. 
Apakah kamu ingin keluar dari Circle dan mencari partner baru?`, roomName), &kb)
		return
	}

	b.room.LeaveRoom(telegramID)

	partnerID, err := b.chat.NextPartner(telegramID)
	if err != nil {
		b.sendMessage(telegramID, fmt.Sprintf("⚠️ %s", err.Error()), nil)
		return
	}

	if partnerID > 0 {
		b.sendMessage(partnerID, "👋 *Partner kamu telah memutus chat.*\n\nGunakan /search untuk mencari partner baru.", nil)
	}

	b.sendMessage(telegramID, "⏭️ *Mencari partner baru...*", nil)
	b.startSearch(telegramID, "", "", 0)
}

func (b *Bot) handleStop(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	state, _, _ := b.db.GetUserState(telegramID)
	if state == models.StateSearching {
		b.chat.CancelSearch(telegramID)
		b.sendMessage(telegramID, "🛑 Pencarian dihentikan.", nil)
		return
	}

	session, _ := b.db.GetActiveSession(telegramID)
	if session == nil {
		b.sendMessageHTML(telegramID, "⚠️ <b>Tidak ada chat aktif saat ini.</b>", nil)
		return
	}

	partnerID, err := b.chat.StopChat(telegramID)
	if err != nil {
		b.sendMessageHTML(telegramID, "❌ Gagal menghentikan chat.", nil)
		return
	}

	duration := time.Since(session.StartedAt).Minutes()
	b.checkChatMarathon(session.User1ID, duration)
	b.checkChatMarathon(session.User2ID, duration)
	b.checkAchievements(session.User1ID)
	b.checkAchievements(session.User2ID)

	b.sendMessageHTML(partnerID, "👋 <b>Partner telah menghentikan chat.</b>", nil)
	b.sendMessageHTML(telegramID, "🛑 <b>Chat dihentikan.</b>\nKetik /search untuk mencari partner baru.", nil)
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
		photoMsg.Caption = "🖼️ *Foto Sekali Lihat* (Akan terhapus dalam 10 detik)"
		if msg.Caption != "" {
			photoMsg.Caption += "\n\n💬 Stranger: " + msg.Caption
		}
		photoMsg.ParseMode = "Markdown"
		sentMsg, _ := b.api.Send(photoMsg)

		go func(chatID int64, messageID int) {
			time.Sleep(10 * time.Second)
			deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
			b.api.Send(deleteMsg)
		}(partnerID, sentMsg.MessageID)
	} else if msg.Voice != nil {
		voice := tgbotapi.NewVoice(partnerID, tgbotapi.FileID(msg.Voice.FileID))
		b.api.Send(voice)
	} else if msg.Video != nil {
		video := tgbotapi.NewVideo(partnerID, tgbotapi.FileID(msg.Video.FileID))
		video.Caption = "📹 *Video Sekali Lihat* (Akan terhapus dalam 15 detik)"
		if msg.Caption != "" {
			video.Caption += "\n\n💬 Stranger: " + msg.Caption
		}
		video.ParseMode = "Markdown"
		sentMsg, _ := b.api.Send(video)

		go func(chatID int64, messageID int) {
			time.Sleep(15 * time.Second)
			deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
			b.api.Send(deleteMsg)
		}(partnerID, sentMsg.MessageID)
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

	header := "<b>📋 Confession Terbaru</b>\n━━━━━━━━━━━━━━━━━━━\n\n"

	for _, c := range confessions {
		emoji := models.DepartmentEmoji(models.Department(c.Department))
		counts, _ := b.confession.GetReactionCounts(c.ID)
		replyCount, _ := b.db.GetConfessionReplyCount(c.ID)

		reactionStr := ""
		for r, count := range counts {
			reactionStr += fmt.Sprintf("%s%d ", r, count)
		}

		replyStr := ""
		if replyCount > 0 {
			replyStr = fmt.Sprintf("💬 %d Replies", replyCount)
		}

		text := fmt.Sprintf(`💬 <b>#%d</b> | %s %s
%s
%s %s
─────────────────
`, c.ID, emoji, html.EscapeString(c.Department), html.EscapeString(c.Content), reactionStr, replyStr)

		header += text
	}

	header += "\n<i>React: ketik</i> /react &lt;id&gt; &lt;emoji&gt;\n<i>Balas: ketik</i> /reply &lt;id&gt; &lt;pesan&gt;\n<i>Lihat: ketik</i> /view_replies &lt;id&gt;"

	b.sendMessageHTML(telegramID, header, nil)
}

func (b *Bot) handleReact(msg *tgbotapi.Message) {
	telegramID := msg.From.ID
	args := msg.CommandArguments()

	if args == "" {
		b.sendMessage(telegramID, "💡 Cara menggunakan: `/react <id> <emoji>`\nContoh: `/react 1 ❤️`", nil)
		return
	}

	parts := strings.Fields(args)
	if len(parts) < 2 {
		b.sendMessage(telegramID, "⚠️ Format salah. Contoh: `/react 1 ❤️`", nil)
		return
	}

	confessionID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		b.sendMessage(telegramID, "⚠️ ID confession harus berupa angka.", nil)
		return
	}

	reaction := parts[1]
	err = b.confession.ReactToConfession(confessionID, telegramID, reaction)
	if err != nil {
		b.sendMessage(telegramID, fmt.Sprintf("❌ %s", err.Error()), nil)
		return
	}

	b.sendMessage(telegramID, fmt.Sprintf("✅ Berhasil menambahkan reaksi %s ke confession #%d", reaction, confessionID), nil)
}

func (b *Bot) handleReply(msg *tgbotapi.Message) {
	telegramID := msg.From.ID
	args := msg.CommandArguments()

	if args == "" {
		b.sendMessage(telegramID, "💡 Cara membalas: `/reply <id> <pesan>`\nContoh: `/reply 1 Semangat ya!`", nil)
		return
	}

	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 {
		b.sendMessage(telegramID, "⚠️ Format salah. Contoh: `/reply 1 Halo!`", nil)
		return
	}

	confessionID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		b.sendMessage(telegramID, "⚠️ ID confession harus berupa angka.", nil)
		return
	}

	content := parts[1]
	err = b.db.CreateConfessionReply(confessionID, telegramID, content)
	if err != nil {
		b.sendMessage(telegramID, "❌ Gagal mengirim balasan.", nil)
		return
	}

	confession, _ := b.db.GetConfession(confessionID)
	if confession != nil && confession.AuthorID != telegramID {
		b.db.IncrementUserKarma(confession.AuthorID, 1)
		b.checkAchievements(confession.AuthorID)
	}

	b.checkAchievements(telegramID)

	b.sendMessage(telegramID, fmt.Sprintf("✅ Berhasil membalas confession #%d", confessionID), nil)
}

func (b *Bot) handleViewReplies(msg *tgbotapi.Message) {
	telegramID := msg.From.ID
	args := msg.CommandArguments()

	if args == "" {
		b.sendMessage(telegramID, "💡 Cara melihat: `/view_replies <id>`", nil)
		return
	}

	confessionID, err := strconv.ParseInt(args, 10, 64)
	if err != nil {
		b.sendMessage(telegramID, "⚠️ ID confession harus berupa angka.", nil)
		return
	}

	confession, err := b.db.GetConfession(confessionID)
	if err != nil || confession == nil {
		b.sendMessage(telegramID, "❌ Confession tidak ditemukan.", nil)
		return
	}

	replies, err := b.db.GetConfessionReplies(confessionID)
	if err != nil {
		b.sendMessage(telegramID, "❌ Gagal mengambil balasan.", nil)
		return
	}

	if len(replies) == 0 {
		b.sendMessage(telegramID, fmt.Sprintf("📋 Belum ada balasan untuk confession #%d.", confessionID), nil)
		return
	}

	response := fmt.Sprintf("<b>📋 Balasan Confession #%d</b>\n", confessionID)
	response += fmt.Sprintf("&gt; <i>%s</i>\n", html.EscapeString(confession.Content))
	response += "━━━━━━━━━━━━━━━━━━━\n\n"

	for i, r := range replies {
		response += fmt.Sprintf("<b>%d.</b> %s\n\n", i+1, html.EscapeString(r.Content))
	}

	b.sendMessageHTML(telegramID, response, nil)
}

func (b *Bot) handlePoll(msg *tgbotapi.Message) {
	telegramID := msg.From.ID
	args := msg.CommandArguments()

	if args == "" {
		b.sendMessageHTML(telegramID, `<b>🗳️ Cara Membuat Polling Anonim</b>

Ketik: <code>/poll Pertanyaan | Opsi 1 | Opsi 2 | ...</code>

Contoh: <code>/poll Setuju gak harga parkir naik? | Setuju | Tidak Setuju</code>`, nil)
		return
	}

	parts := strings.Split(args, "|")
	if len(parts) < 3 {
		b.sendMessageHTML(telegramID, "⚠️ <b>Format salah.</b> Minimal harus ada pertanyaan dan 2 opsi jawaban.", nil)
		return
	}

	question := strings.TrimSpace(parts[0])
	var options []string
	for i := 1; i < len(parts); i++ {
		opt := strings.TrimSpace(parts[i])
		if opt != "" {
			options = append(options, opt)
		}
	}

	if len(options) < 2 {
		b.sendMessageHTML(telegramID, "⚠️ <b>Format salah.</b> Minimal harus ada 2 opsi jawaban yang valid.", nil)
		return
	}

	pollID, err := b.db.CreatePoll(telegramID, question, options)
	if err != nil {
		b.sendMessageHTML(telegramID, "❌ Gagal membuat polling.", nil)
		return
	}

	b.db.IncrementUserKarma(telegramID, 3)
	b.checkAchievements(telegramID)

	b.sendMessageHTML(telegramID, fmt.Sprintf("✅ <b>Polling #%d berhasil dibuat!</b>\nSemua mahasiswa sekarang bisa memberikan suara secara anonim.", pollID), nil)
}

func (b *Bot) handleViewPolls(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	polls, err := b.db.GetLatestPolls(15)
	if err != nil {
		b.sendMessageHTML(telegramID, "❌ Gagal mengambil polling.", nil)
		return
	}

	if len(polls) == 0 {
		b.sendMessageHTML(telegramID, "📋 Belum ada polling aktif. Buat yang pertama dengan /poll!", nil)
		return
	}

	header := "<b>🗳️ Daftar Polling Terbaru</b>\n━━━━━━━━━━━━━━━━━━━\n\n"

	for _, p := range polls {
		count, _ := b.db.GetPollVoteCount(p.ID)
		header += fmt.Sprintf("📊 <b>#%d</b>: %s\n👥 <i>%d Suara</i>\n\n", p.ID, html.EscapeString(p.Question), count)
	}

	header += "━━━━━━━━━━━━━━━━━━━\n<i>Ikut memilih: ketik</i> <code>/vote_poll &lt;id&gt;</code>"

	b.sendMessageHTML(telegramID, header, nil)
}

func (b *Bot) handleVotePoll(msg *tgbotapi.Message) {
	telegramID := msg.From.ID
	args := msg.CommandArguments()

	if args == "" {
		b.sendMessageHTML(telegramID, "💡 Cara memilih: <code>/vote_poll &lt;id&gt;</code>", nil)
		return
	}

	pollID, err := strconv.ParseInt(args, 10, 64)
	if err != nil {
		b.sendMessageHTML(telegramID, "⚠️ ID polling harus berupa angka.", nil)
		return
	}

	p, err := b.db.GetPoll(pollID)
	if err != nil || p == nil {
		b.sendMessageHTML(telegramID, "❌ Polling tidak ditemukan.", nil)
		return
	}

	kb := PollVoteKeyboard(p.ID, p.Options)
	text := fmt.Sprintf("🗳️ <b>Polling #%d</b>\n\n<b>Pertanyaan:</b>\n%s\n\n<i>Pilih opsi di bawah untuk memberikan suara secara anonim:</i>", p.ID, html.EscapeString(p.Question))
	b.sendMessageHTML(telegramID, text, &kb)
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

	b.sendMessageHTML(telegramID, fmt.Sprintf("✅ <b>Whisper Terkirim!</b>\n\n📤 Dikirim ke <b>%d</b> mahasiswa %s %s",
		len(targets),
		models.DepartmentEmoji(models.Department(targetDept)),
		html.EscapeString(targetDept),
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

	earned, _ := b.db.GetUserAchievements(telegramID)
	badgeStr := ""
	if len(earned) > 0 {
		badgeStr = "\n🏆 <b>Lencana:</b> "
		allAch := models.GetAchievements()
		for _, ua := range earned {
			if ach, ok := allAch[ua.AchievementKey]; ok {
				badgeStr += ach.Icon + " "
			}
		}
		badgeStr += "\n"
	}

	profileText := fmt.Sprintf(`<b>👤 Profil Kamu</b>

━━━━━━━━━━━━━━━━━━━
🏷️ <b>Nama Anonim:</b> %s
✨ <b>Karma:</b> <b>%d</b>
%s <b>Gender:</b> %s
🎓 <b>Angkatan:</b> %d
%s <b>Jurusan:</b> %s
📧 <b>Email:</b> %s
━━━━━━━━━━━━━━━━━━━
📊 <b>Statistik:</b>
💬 Total Chat: <b>%d</b>
📝 Confessions: <b>%d</b>
❤️ Reactions Diterima: <b>%d</b>
📅 Hari Aktif: <b>%d</b>%s
━━━━━━━━━━━━━━━━━━━
⚠️ Report Count: %d/3`,
		html.EscapeString(user.DisplayName),
		user.Karma,
		models.GenderEmoji(user.Gender), html.EscapeString(string(user.Gender)),
		user.Year,
		models.DepartmentEmoji(user.Department), html.EscapeString(string(user.Department)),
		html.EscapeString(maskEmail(user.Email)),
		totalChats,
		totalConfessions,
		totalReactions,
		daysSince,
		badgeStr,
		user.ReportCount,
	)

	kb := BackToMenuKeyboard()
	b.sendMessageHTML(telegramID, profileText, &kb)
}

func (b *Bot) handleStats(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	user, err := b.db.GetUser(telegramID)
	if err != nil || user == nil {
		b.sendMessage(telegramID, "❌ Gagal memuat profil.", nil)
		return
	}

	totalChats, totalConfessions, totalReactions, daysSince, err := b.profile.GetStats(telegramID)
	if err != nil {
		b.sendMessage(telegramID, "❌ Gagal memuat statistik.", nil)
		return
	}

	statsText := fmt.Sprintf(`<b>📊 Statistik Kamu</b>

━━━━━━━━━━━━━━━━━━━
✨ Total Karma: <b>%d</b>
💬 Total Chat: <b>%d</b>
📝 Confession Dibuat: <b>%d</b>
❤️ Reactions Diterima: <b>%d</b>
📅 Hari Sejak Bergabung: <b>%d</b>
━━━━━━━━━━━━━━━━━━━

<i>Terus berinteraksi untuk meningkatkan statistik kamu!</i> 🚀`,
		user.Karma, totalChats, totalConfessions, totalReactions, daysSince)

	kb := BackToMenuKeyboard()
	b.sendMessageHTML(telegramID, statsText, &kb)
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

func (b *Bot) handleCircles(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	rooms, err := b.room.GetActiveRooms()
	if err != nil {
		b.sendMessage(telegramID, "❌ Gagal mengambil daftar circle.", nil)
		return
	}

	kb := RoomsKeyboard(rooms)
	text := `👥 <b>Anonymous Circles (Group Chat)</b>

━━━━━━━━━━━━━━━━━━━
Gabung ke circle topik tertentu untuk ngobrol bareng mahasiswa lainnya secara anonim!

📌 <b>Pilih Circle:</b>`

	b.sendMessageHTML(telegramID, text, &kb)
}

func (b *Bot) handleLeaveCircle(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	room, err := b.room.GetUserRoom(telegramID)
	if err != nil || room == nil {
		b.sendMessage(telegramID, "⚠️ Kamu tidak sedang berada di circle mana pun.", nil)
		return
	}

	err = b.room.LeaveRoom(telegramID)
	if err != nil {
		b.sendMessage(telegramID, "❌ Gagal keluar dari circle.", nil)
		return
	}

	b.sendMessageHTML(telegramID, fmt.Sprintf("👋 <b>Kamu telah keluar dari circle %s</b>", room.Name), nil)
	b.showMainMenu(telegramID, nil)
}

func (b *Bot) handleCircleMessage(msg *tgbotapi.Message) {
	telegramID := msg.From.ID

	members, roomName, err := b.room.GetRoomMembers(telegramID)
	if err != nil {
		b.db.SetUserState(telegramID, models.StateNone, "")
		b.sendMessage(telegramID, "⚠️ Kamu tidak berada di circle aktif. Gunakan /circles untuk bergabung.", nil)
		return
	}

	user, _ := b.db.GetUser(telegramID)
	senderInfo := "Anonymous"
	if user != nil {
		senderInfo = fmt.Sprintf("%s %s", models.GenderEmoji(user.Gender), string(user.Department))
	}

	broadcastText := fmt.Sprintf("👥 <b>[%s]</b>\n👤 %s: %s", roomName, senderInfo, html.EscapeString(msg.Text))

	for _, memberID := range members {
		if memberID == telegramID {
			continue
		}

		if msg.Text != "" {
			b.sendMessageHTML(memberID, broadcastText, nil)
		} else {
			b.forwardMedia(memberID, msg, fmt.Sprintf("👥 [%s] 👤 %s", roomName, senderInfo))
		}
	}
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

func (b *Bot) handleRoomNameInput(msg *tgbotapi.Message) {
	telegramID := msg.From.ID
	name := strings.TrimSpace(msg.Text)

	if name == "" || len(name) < 3 {
		b.sendMessage(telegramID, "⚠️ Nama circle terlalu pendek. Minimal 3 karakter.", nil)
		return
	}

	if len(name) > 30 {
		b.sendMessage(telegramID, "⚠️ Nama circle terlalu panjang. Maksimal 30 karakter.", nil)
		return
	}

	b.db.SetUserState(telegramID, models.StateAwaitingRoomDesc, name)
	b.sendMessage(telegramID, fmt.Sprintf("📝 *Nama Circle:* %s\n\nSekarang tulis *Deskripsi Singkat* untuk circle ini:", name), nil)
}

func (b *Bot) handleRoomDescInput(msg *tgbotapi.Message) {
	telegramID := msg.From.ID
	desc := strings.TrimSpace(msg.Text)
	_, name, _ := b.db.GetUserState(telegramID)

	if desc == "" || len(desc) < 5 {
		b.sendMessage(telegramID, "⚠️ Deskripsi terlalu pendek. Minimal 5 karakter.", nil)
		return
	}

	room, err := b.room.CreateRoom(name, desc)
	if err != nil {
		b.sendMessage(telegramID, fmt.Sprintf("❌ %s", err.Error()), nil)
		b.db.SetUserState(telegramID, models.StateNone, "")
		return
	}

	b.db.SetUserState(telegramID, models.StateNone, "")
	b.sendMessageHTML(telegramID, fmt.Sprintf("✅ <b>Circle Berhasil Dibuat!</b>\n\nSekarang kamu dan orang lain bisa bergabung ke <b>%s</b> melalui menu /circles.", room.Name), nil)

	b.room.JoinRoom(telegramID, room.Slug)

	kb := LeaveCircleKeyboard()
	b.sendMessageHTML(telegramID, fmt.Sprintf("🎉 Kamu otomatis bergabung ke circle <b>%s</b>. Selamat ngobrol!", room.Name), &kb)
}
