package service

import (
	"context"
	"testing"
)

func TestGamificationRewardActivity(t *testing.T) {
	db := setupTestDB(t)
	gamificationSvc := NewGamificationService(db)
	ctx := context.Background()
	userID := int64(6001)

	createUserForTest(t, db, userID, "", "", 0)

	level, leveledUp, points, exp, err := gamificationSvc.RewardActivity(ctx, userID, "chat_message")
	if err != nil {
		t.Fatalf("RewardActivity failed: %v", err)
	}
	if points != 1 || exp != 5 || level != 1 || leveledUp {
		t.Errorf("Unexpected rewards for first chat: points=%d, exp=%d, level=%d", points, exp, level)
	}

	_, leveledUp, _, _, _ = gamificationSvc.RewardActivity(ctx, userID, "confession_created")
	if leveledUp {
		t.Errorf("Should not level up yet")
	}

	level, leveledUp, _, _, _ = gamificationSvc.RewardActivity(ctx, userID, "confession_created")
	if !leveledUp || level != 2 {
		t.Errorf("Expected level up to 2, got level %d, leveledUp %v", level, leveledUp)
	}

	user, _ := db.GetUser(ctx, userID)
	if user.Points != 21 {
		t.Errorf("Expected 21 points, got %d", user.Points)
	}
}

func TestGamificationDailyStreak(t *testing.T) {
	db := setupTestDB(t)
	gamificationSvc := NewGamificationService(db)
	ctx := context.Background()
	userID := int64(6002)

	createUserForTest(t, db, userID, "", "", 0)

	streak, bonus, err := gamificationSvc.UpdateStreak(ctx, userID)
	if err != nil {
		t.Fatalf("UpdateStreak failed: %v", err)
	}
	if streak != 1 || bonus {
		t.Errorf("Expected streak 1, no bonus, got streak %d, bonus %v", streak, bonus)
	}
}

func TestGamificationLeaderboard(t *testing.T) {
	db := setupTestDB(t)
	gamificationSvc := NewGamificationService(db)
	ctx := context.Background()

	for i := int64(7001); i <= 7005; i++ {
		createUserForTest(t, db, i, "", "", 0)
		_ = db.UpdateUserVerified(ctx, i, true)
		_, _, _, _, _ = gamificationSvc.RewardActivity(ctx, i, "chat_message")
	}

	leaderboard, err := gamificationSvc.GetLeaderboard(ctx)
	if err != nil {
		t.Fatalf("GetLeaderboard failed: %v", err)
	}
	if len(leaderboard) != 5 {
		t.Fatalf("Expected 5 in leaderboard, got %d", len(leaderboard))
	}
}
