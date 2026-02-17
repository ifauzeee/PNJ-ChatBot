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
	MinEntryYear        = 2018
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

func CurrentEntryYear() int {
	return time.Now().Year()
}

func IsValidEntryYear(year int) bool {
	return year >= MinEntryYear && year <= CurrentEntryYear()
}

func AvailableEntryYears() []int {
	currentYear := CurrentEntryYear()
	if currentYear < MinEntryYear {
		return []int{MinEntryYear}
	}

	years := make([]int, 0, currentYear-MinEntryYear+1)
	for year := currentYear; year >= MinEntryYear; year-- {
		years = append(years, year)
	}

	return years
}

type User struct {
	ID           int64      `json:"id" db:"id"`
	TelegramID   int64      `json:"telegram_id" db:"telegram_id"`
	Email        string     `json:"email" db:"email"`
	Gender       Gender     `json:"gender" db:"gender"`
	Department   Department `json:"department" db:"department"`
	Year         int        `json:"year" db:"year"`
	DisplayName  string     `json:"display_name" db:"display_name"`
	Karma        int        `json:"karma" db:"karma"`
	IsVerified   bool       `json:"is_verified" db:"is_verified"`
	IsBanned     bool       `json:"is_banned" db:"is_banned"`
	ReportCount  int        `json:"report_count" db:"report_count"`
	TotalChats   int        `json:"total_chats" db:"total_chats"`
	Points       int        `json:"points" db:"points"`
	Level        int        `json:"level" db:"level"`
	Exp          int        `json:"exp" db:"exp"`
	DailyStreak  int        `json:"daily_streak" db:"daily_streak"`
	LastActiveAt time.Time  `json:"last_active_at" db:"last_active_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

type UserState string

const (
	StateNone                UserState = ""
	StateAwaitingEmail       UserState = "awaiting_email"
	StateAwaitingOTP         UserState = "awaiting_otp"
	StateAwaitingGender      UserState = "awaiting_gender"
	StateAwaitingDept        UserState = "awaiting_department"
	StateAwaitingYear        UserState = "awaiting_year"
	StateSearching           UserState = "searching"
	StateInChat              UserState = "in_chat"
	StateAwaitingConfess     UserState = "awaiting_confession"
	StateAwaitingReport      UserState = "awaiting_report"
	StateAwaitingWhisper     UserState = "awaiting_whisper"
	StateAwaitingWhisperDept UserState = "awaiting_whisper_dept"
	StateAwaitingEditField   UserState = "awaiting_edit_field"
	StateInCircle            UserState = "in_circle"
	StateAwaitingRoomName    UserState = "awaiting_room_name"
	StateAwaitingRoomDesc    UserState = "awaiting_room_desc"
)

type Room struct {
	ID          int64     `json:"id" db:"id"`
	Slug        string    `json:"slug" db:"slug"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	MemberCount int       `json:"member_count" db:"member_count"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type RoomMember struct {
	ID         int64     `json:"id" db:"id"`
	RoomID     int64     `json:"room_id" db:"room_id"`
	TelegramID int64     `json:"telegram_id" db:"telegram_id"`
	JoinedAt   time.Time `json:"joined_at" db:"joined_at"`
}

type VerificationCode struct {
	ID         int64     `json:"id" db:"id"`
	TelegramID int64     `json:"telegram_id" db:"telegram_id"`
	Email      string    `json:"email" db:"email"`
	Code       string    `json:"code" db:"code"`
	ExpiresAt  time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

type ChatSession struct {
	ID        int64      `json:"id" db:"id"`
	User1ID   int64      `json:"user1_id" db:"user1_id"`
	User2ID   int64      `json:"user2_id" db:"user2_id"`
	IsActive  bool       `json:"is_active" db:"is_active"`
	StartedAt time.Time  `json:"started_at" db:"started_at"`
	EndedAt   *time.Time `json:"ended_at" db:"ended_at"`
}

type Confession struct {
	ID         int64     `json:"id" db:"id"`
	AuthorID   int64     `json:"author_id" db:"author_id"`
	Content    string    `json:"content" db:"content"`
	LikeCount  int       `json:"like_count" db:"like_count"`
	Department string    `json:"department" db:"department"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

type ConfessionReaction struct {
	ID           int64     `json:"id" db:"id"`
	ConfessionID int64     `json:"confession_id" db:"confession_id"`
	TelegramID   int64     `json:"telegram_id" db:"telegram_id"`
	Reaction     string    `json:"reaction" db:"reaction"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type Report struct {
	ID            int64     `json:"id" db:"id"`
	ReporterID    int64     `json:"reporter_id" db:"reporter_id"`
	ReportedID    int64     `json:"reported_id" db:"reported_id"`
	Reason        string    `json:"reason" db:"reason"`
	Evidence      string    `json:"evidence" db:"evidence"`
	ChatSessionID int64     `json:"chat_session_id" db:"chat_session_id"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

type BlockedUser struct {
	ID        int64     `json:"id" db:"id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	BlockedID int64     `json:"blocked_id" db:"blocked_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type UserStats struct {
	TotalChats       int `json:"total_chats"`
	TotalConfessions int `json:"total_confessions"`
	TotalReactions   int `json:"total_reactions"`
	DaysActive       int `json:"days_active"`
}

type Whisper struct {
	ID           int64     `json:"id" db:"id"`
	SenderID     int64     `json:"sender_id" db:"sender_id"`
	TargetDept   string    `json:"target_dept" db:"target_dept"`
	Content      string    `json:"content" db:"content"`
	SenderDept   string    `json:"sender_dept" db:"sender_dept"`
	SenderGender string    `json:"sender_gender" db:"sender_gender"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type ConfessionReply struct {
	ID           int64     `json:"id" db:"id"`
	ConfessionID int64     `json:"confession_id" db:"confession_id"`
	AuthorID     int64     `json:"author_id" db:"author_id"`
	Content      string    `json:"content" db:"content"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type Poll struct {
	ID        int64         `json:"id" db:"id"`
	AuthorID  int64         `json:"author_id" db:"author_id"`
	Question  string        `json:"question" db:"question"`
	Options   []*PollOption `json:"options" db:"-"`
	CreatedAt time.Time     `json:"created_at" db:"created_at"`
}

type PollOption struct {
	ID         int64  `json:"id" db:"id"`
	PollID     int64  `json:"poll_id" db:"poll_id"`
	OptionText string `json:"option_text" db:"option_text"`
	VoteCount  int    `json:"vote_count" db:"vote_count"`
}

type Achievement struct {
	Key         string
	Name        string
	Description string
	Icon        string
}

func GetAchievements() map[string]Achievement {
	return map[string]Achievement{
		"CHAT_MARATHON": {
			Key:         "CHAT_MARATHON",
			Name:        "Chat Marathon",
			Description: "Chatting selama lebih dari 1 jam tanpa henti.",
			Icon:        "🏃",
		},
		"POPULAR_AUTHOR": {
			Key:         "POPULAR_AUTHOR",
			Name:        "Penulis Populer",
			Description: "Confession mendapatkan lebih dari 5 reaksi.",
			Icon:        "🌟",
		},
		"KARMA_MASTER": {
			Key:         "KARMA_MASTER",
			Name:        "Karma Master",
			Description: "Mencapai lebih dari 50 poin Karma.",
			Icon:        "👑",
		},
		"POLL_MAKER": {
			Key:         "POLL_MAKER",
			Name:        "Pembuat Aspirasi",
			Description: "Membuat lebih dari 3 polling.",
			Icon:        "🗳️",
		},
	}
}

type UserAchievement struct {
	TelegramID     int64     `db:"telegram_id"`
	AchievementKey string    `db:"achievement_key"`
	EarnedAt       time.Time `db:"earned_at"`
}
