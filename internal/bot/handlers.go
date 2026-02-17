package bot

import (
	"fmt"
	"html"

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
