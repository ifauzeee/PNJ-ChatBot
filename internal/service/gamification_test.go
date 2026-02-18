package service

import (
	"testing"
)

func TestGamificationRewardActivity(t *testing.T) {
	db := setupTestDB(t)
	gamificationSvc := NewGamificationService(db)
	userID := int64(6001)

	createUserForTest(t, db, userID, "", "", 0)

	level, leveledUp, points, exp, err := gamificationSvc.RewardActivity(userID, "chat_message")
	if err != nil {
		t.Fatalf("RewardActivity failed: %v", err)
	}
	if points != 1 || exp != 5 || level != 1 || leveledUp {
		t.Errorf("Unexpected rewards for first chat: points=%d, exp=%d, level=%d", points, exp, level)
	}

	_, leveledUp, _, _, _ = gamificationSvc.RewardActivity(userID, "confession_created")
	if leveledUp {
		t.Errorf("Should not level up yet")
	}

	level, leveledUp, _, _, _ = gamificationSvc.RewardActivity(userID, "confession_created")
	if !leveledUp || level != 2 {
		t.Errorf("Expected level up to 2, got level %d, leveledUp %v", level, leveledUp)
	}

	user, _ := db.GetUser(userID)
	if user.Points != 21 {
		t.Errorf("Expected 21 points, got %d", user.Points)
	}
}

func TestGamificationDailyStreak(t *testing.T) {
	db := setupTestDB(t)
	gamificationSvc := NewGamificationService(db)
	userID := int64(7001)

	createUserForTest(t, db, userID, "", "", 0)

	streak, bonus, err := gamificationSvc.UpdateStreak(userID)
	if err != nil {
		t.Fatalf("UpdateStreak failed: %v", err)
	}
	if streak != 1 || bonus {
		t.Errorf("Initial streak should be 1, no bonus. Got streak=%d, bonus=%v", streak, bonus)
	}

	streak, bonus, _ = gamificationSvc.UpdateStreak(userID)
	if streak != 1 || bonus {
		t.Errorf("Same day streak should stay 1. Got streak=%d, bonus=%v", streak, bonus)
	}
}

func TestLeaderboard(t *testing.T) {
	db := setupTestDB(t)
	gamificationSvc := NewGamificationService(db)

	createUserForTest(t, db, 8001, "", "", 0)
	createUserForTest(t, db, 8002, "", "", 0)

	_ = db.UpdateUserVerified(8001, true)
	_ = db.UpdateUserVerified(8002, true)
	_ = db.UpdateUserDisplayName(8001, "User A")
	_ = db.UpdateUserDisplayName(8002, "User B")

	_, _, _ = db.AddPointsAndExp(8001, 100, 500)
	_, _, _ = db.AddPointsAndExp(8002, 50, 250)

	leaderboard, err := gamificationSvc.GetLeaderboard()
	if err != nil {
		t.Fatalf("GetLeaderboard failed: %v", err)
	}

	if len(leaderboard) != 2 {
		t.Fatalf("Expected 2 users in leaderboard, got %d", len(leaderboard))
	}

	if leaderboard[0].DisplayName != "User A" {
		t.Errorf("Expected User A at rank 1, got %s", leaderboard[0].DisplayName)
	}
}
