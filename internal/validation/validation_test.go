package validation

import (
	"strings"
	"testing"
)

func TestValidateTextTooShort(t *testing.T) {
	result := ValidateText("ab", TextLimits{MinLen: 5, MaxLen: 100, Label: "Test"})
	if result == "" {
		t.Fatal("Expected error for short text, got empty")
	}
	if !strings.Contains(result, "terlalu pendek") {
		t.Fatalf("Expected 'terlalu pendek' in error, got: %s", result)
	}
}

func TestValidateTextTooLong(t *testing.T) {
	text := strings.Repeat("a", 101)
	result := ValidateText(text, TextLimits{MinLen: 1, MaxLen: 100, Label: "Test"})
	if result == "" {
		t.Fatal("Expected error for long text, got empty")
	}
	if !strings.Contains(result, "terlalu panjang") {
		t.Fatalf("Expected 'terlalu panjang' in error, got: %s", result)
	}
}

func TestValidateTextValid(t *testing.T) {
	result := ValidateText("Hello World!", TextLimits{MinLen: 1, MaxLen: 100, Label: "Test"})
	if result != "" {
		t.Fatalf("Expected no error, got: %s", result)
	}
}

func TestValidateTextUnicodeLength(t *testing.T) {

	result := ValidateText("Halo 🎭", TextLimits{MinLen: 1, MaxLen: 10, Label: "Test"})
	if result != "" {
		t.Fatalf("Expected valid for unicode text, got: %s", result)
	}
}

func TestSanitizeText(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  hello  ", "hello"},
		{"hello\x00world", "helloworld"},
		{"line1\nline2", "line1\nline2"},
		{"tab\there", "tab\there"},
		{"\x01\x02test", "test"},
	}

	for _, tt := range tests {
		result := SanitizeText(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeText(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}


func TestContainsOnlyPrintable(t *testing.T) {
	if !ContainsOnlyPrintable("Hello World\nNew line") {
		t.Error("Expected true for normal text")
	}
	if ContainsOnlyPrintable("Hello\x00World") {
		t.Error("Expected false for text with null byte")
	}
}

func TestPredefinedLimits(t *testing.T) {

	limits := []TextLimits{
		ConfessionLimits, WhisperLimits, RoomNameLimits,
		RoomDescLimits, ReportLimits, ReplyLimits,
		PollQuestionLimits, PollOptionLimits,
	}

	for _, l := range limits {
		if l.MinLen <= 0 {
			t.Errorf("%s: MinLen must be > 0, got %d", l.Label, l.MinLen)
		}
		if l.MaxLen < l.MinLen {
			t.Errorf("%s: MaxLen (%d) must be >= MinLen (%d)", l.Label, l.MaxLen, l.MinLen)
		}
		if l.Label == "" {
			t.Error("Label must not be empty")
		}
	}
}
