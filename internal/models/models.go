package models

import "time"

type Department string

const (
	DeptTeknikSipil       Department = "Teknik Sipil"
	DeptTeknikMesin       Department = "Teknik Mesin"
	DeptTeknikElektro     Department = "Teknik Elektro"
	DeptTeknikInformatika Department = "Teknik Informatika & Komputer"
	DeptTeknikGrafika     Department = "Teknik Grafika & Penerbitan"
	DeptAkuntansi         Department = "Akuntansi"
	DeptAdministrasiNiaga Department = "Administrasi Niaga"
	DeptPascasarjana      Department = "Pascasarjana"
)

func AllDepartments() []Department {
	return []Department{
		DeptTeknikSipil,
		DeptTeknikMesin,
		DeptTeknikElektro,
		DeptTeknikInformatika,
		DeptTeknikGrafika,
		DeptAkuntansi,
		DeptAdministrasiNiaga,
		DeptPascasarjana,
	}
}

func DepartmentEmoji(dept Department) string {
	switch dept {
	case DeptTeknikSipil:
		return "🏗️"
	case DeptTeknikMesin:
		return "⚙️"
	case DeptTeknikElektro:
		return "⚡"
	case DeptTeknikInformatika:
		return "💻"
	case DeptTeknikGrafika:
		return "🎨"
	case DeptAkuntansi:
		return "📊"
	case DeptAdministrasiNiaga:
		return "📈"
	case DeptPascasarjana:
		return "🎓"
	default:
		return "📚"
	}
}

func IsValidDepartment(dept string) bool {
	for _, d := range AllDepartments() {
		if string(d) == dept {
			return true
		}
	}
	return false
}

type Gender string

const (
	GenderMale   Gender = "Laki-laki"
	GenderFemale Gender = "Perempuan"
)

func GenderEmoji(g Gender) string {
	switch g {
	case GenderMale:
		return "👨"
	case GenderFemale:
		return "👩"
	default:
		return "🧑"
	}
}

func IsValidGender(g string) bool {
	return g == string(GenderMale) || g == string(GenderFemale)
}

type User struct {
	ID          int64      `json:"id"`
	TelegramID  int64      `json:"telegram_id"`
	Email       string     `json:"email"`
	Gender      Gender     `json:"gender"`
	Department  Department `json:"department"`
	DisplayName string     `json:"display_name"`
	IsVerified  bool       `json:"is_verified"`
	IsBanned    bool       `json:"is_banned"`
	ReportCount int        `json:"report_count"`
	TotalChats  int        `json:"total_chats"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type UserState string

const (
	StateNone                UserState = ""
	StateAwaitingEmail       UserState = "awaiting_email"
	StateAwaitingOTP         UserState = "awaiting_otp"
	StateAwaitingGender      UserState = "awaiting_gender"
	StateAwaitingDept        UserState = "awaiting_department"
	StateSearching           UserState = "searching"
	StateInChat              UserState = "in_chat"
	StateAwaitingConfess     UserState = "awaiting_confession"
	StateAwaitingReport      UserState = "awaiting_report"
	StateAwaitingWhisper     UserState = "awaiting_whisper"
	StateAwaitingWhisperDept UserState = "awaiting_whisper_dept"
	StateAwaitingEditField   UserState = "awaiting_edit_field"
)

type VerificationCode struct {
	ID         int64     `json:"id"`
	TelegramID int64     `json:"telegram_id"`
	Email      string    `json:"email"`
	Code       string    `json:"code"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
}

type ChatSession struct {
	ID        int64      `json:"id"`
	User1ID   int64      `json:"user1_id"`
	User2ID   int64      `json:"user2_id"`
	IsActive  bool       `json:"is_active"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at"`
}

type ChatQueue struct {
	ID              int64     `json:"id"`
	TelegramID      int64     `json:"telegram_id"`
	PreferredDept   string    `json:"preferred_dept"`
	PreferredGender string    `json:"preferred_gender"`
	JoinedAt        time.Time `json:"joined_at"`
}

type Confession struct {
	ID         int64     `json:"id"`
	AuthorID   int64     `json:"author_id"`
	Content    string    `json:"content"`
	LikeCount  int       `json:"like_count"`
	Department string    `json:"department"`
	CreatedAt  time.Time `json:"created_at"`
}

type ConfessionReaction struct {
	ID           int64     `json:"id"`
	ConfessionID int64     `json:"confession_id"`
	TelegramID   int64     `json:"telegram_id"`
	Reaction     string    `json:"reaction"`
	CreatedAt    time.Time `json:"created_at"`
}

type Report struct {
	ID            int64     `json:"id"`
	ReporterID    int64     `json:"reporter_id"`
	ReportedID    int64     `json:"reported_id"`
	Reason        string    `json:"reason"`
	ChatSessionID int64     `json:"chat_session_id"`
	CreatedAt     time.Time `json:"created_at"`
}

type BlockedUser struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	BlockedID int64     `json:"blocked_id"`
	CreatedAt time.Time `json:"created_at"`
}

type UserStats struct {
	TotalChats       int `json:"total_chats"`
	TotalConfessions int `json:"total_confessions"`
	TotalReactions   int `json:"total_reactions"`
	DaysActive       int `json:"days_active"`
}

type Whisper struct {
	ID           int64     `json:"id"`
	SenderID     int64     `json:"sender_id"`
	TargetDept   string    `json:"target_dept"`
	Content      string    `json:"content"`
	SenderDept   string    `json:"sender_dept"`
	SenderGender string    `json:"sender_gender"`
	CreatedAt    time.Time `json:"created_at"`
}
