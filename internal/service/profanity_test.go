package service

import "testing"

func TestProfanityServiceDetectsBadWords(t *testing.T) {
	svc := NewProfanityService()

	tests := []struct {
		input    string
		expected bool
	}{
		{"Hai, apa kabar?", false},
		{"Kamu anjing banget!", true},
		{"Dasar goblok!", true},
		{"Ini pesan normal tanpa kata kasar", false},
		{"semua orang tolol", true},
		{"fuck this bot", true},
		{"Bangsat lo!", true},
		{"selamat pagi semua, ayo semangat hari ini", false},
	}

	for _, tt := range tests {
		got := svc.IsBad(tt.input)
		if got != tt.expected {
			t.Errorf("IsBad(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestProfanityServiceClean(t *testing.T) {
	svc := NewProfanityService()

	tests := []struct {
		input   string
		hasStar bool
	}{
		{"Hai, apa kabar?", false},
		{"Kamu anjing banget!", true},
		{"Dasar goblok!", true},
	}

	for _, tt := range tests {
		cleaned := svc.Clean(tt.input)
		if tt.hasStar {
			if cleaned == tt.input {
				t.Errorf("Clean(%q) should have replaced bad words, got %q", tt.input, cleaned)
			}
		} else {
			if cleaned != tt.input {
				t.Errorf("Clean(%q) should not change, got %q", tt.input, cleaned)
			}
		}
	}
}

func TestProfanityServiceCaseInsensitive(t *testing.T) {
	svc := NewProfanityService()

	if !svc.IsBad("ANJING") {
		t.Error("Expected case-insensitive bad word detection for uppercase")
	}
	if !svc.IsBad("Goblok") {
		t.Error("Expected case-insensitive bad word detection for mixed case")
	}
}
