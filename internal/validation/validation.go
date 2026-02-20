package validation

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type TextLimits struct {
	MinLen int
	MaxLen int
	Label  string
}

var (
	ConfessionLimits   = TextLimits{MinLen: 10, MaxLen: 1000, Label: "Confession"}
	WhisperLimits      = TextLimits{MinLen: 5, MaxLen: 500, Label: "Whisper"}
	RoomNameLimits     = TextLimits{MinLen: 3, MaxLen: 30, Label: "Nama Circle"}
	RoomDescLimits     = TextLimits{MinLen: 5, MaxLen: 200, Label: "Deskripsi Circle"}
	ReportLimits       = TextLimits{MinLen: 5, MaxLen: 500, Label: "Alasan Laporan"}
	ReplyLimits        = TextLimits{MinLen: 1, MaxLen: 500, Label: "Balasan"}
	PollQuestionLimits = TextLimits{MinLen: 5, MaxLen: 300, Label: "Pertanyaan Polling"}
	PollOptionLimits   = TextLimits{MinLen: 1, MaxLen: 100, Label: "Opsi Polling"}
	BroadcastLimits    = TextLimits{MinLen: 1, MaxLen: 4000, Label: "Broadcast"}
)

func ValidateText(text string, limits TextLimits) string {
	length := utf8.RuneCountInString(strings.TrimSpace(text))

	if length < limits.MinLen {
		return fmt.Sprintf("⚠️ %s terlalu pendek. Minimal %d karakter.", limits.Label, limits.MinLen)
	}
	if length > limits.MaxLen {
		return fmt.Sprintf("⚠️ %s terlalu panjang. Maksimal %d karakter.", limits.Label, limits.MaxLen)
	}
	return ""
}

func SanitizeText(text string) string {
	text = strings.TrimSpace(text)

	cleaned := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' || r == '\r' {
			return r
		}
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, text)
	return cleaned
}

func ContainsOnlyPrintable(text string) bool {
	for _, r := range text {
		if r < 32 && r != '\n' && r != '\t' && r != '\r' {
			return false
		}
	}
	return true
}
