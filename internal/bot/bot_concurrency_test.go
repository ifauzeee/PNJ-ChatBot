package bot

import (
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestExtractUpdateUserID(t *testing.T) {
	b := &Bot{}

	id, ok := b.extractUpdateUserID(tgbotapi.Update{
		CallbackQuery: &tgbotapi.CallbackQuery{
			From: &tgbotapi.User{ID: 123},
		},
	})
	if !ok || id != 123 {
		t.Fatalf("expected callback user id 123, got id=%d ok=%v", id, ok)
	}

	id, ok = b.extractUpdateUserID(tgbotapi.Update{
		Message: &tgbotapi.Message{
			From: &tgbotapi.User{ID: 456},
		},
	})
	if !ok || id != 456 {
		t.Fatalf("expected message user id 456, got id=%d ok=%v", id, ok)
	}

	_, ok = b.extractUpdateUserID(tgbotapi.Update{})
	if ok {
		t.Fatalf("expected false for update without user context")
	}
}

func TestGetUserLockStablePerUser(t *testing.T) {
	b := &Bot{}

	lock1 := b.getUserLock(1)
	lock1Again := b.getUserLock(1)
	lock2 := b.getUserLock(2)

	if lock1 != lock1Again {
		t.Fatalf("expected same lock pointer for same user")
	}
	if lock1 == lock2 {
		t.Fatalf("expected different lock pointer for different users")
	}
}
