package service

import (
	"context"
	"testing"

	"github.com/pnj-anonymous-bot/internal/config"
)

func TestConfessionServiceCreateAndGet(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.Config{MaxConfessionsPerHour: 3}
	confessionSvc := NewConfessionService(db, cfg)
	ctx := context.Background()
	userID := int64(10001)

	createUserForTest(t, db, userID, "Laki-laki", "Teknik Informatika & Komputer", 2022)

	confession, err := confessionSvc.CreateConfession(ctx, userID, "Ini adalah confession pertama saya yang cukup panjang.")
	if err != nil {
		t.Fatalf("CreateConfession failed: %v", err)
	}
	if confession == nil {
		t.Fatal("Expected non-nil confession")
	}
	if confession.AuthorID != userID {
		t.Errorf("Expected author ID %d, got %d", userID, confession.AuthorID)
	}
	if confession.Department != "Teknik Informatika & Komputer" {
		t.Errorf("Expected TIK department, got %s", confession.Department)
	}

	fetched, err := confessionSvc.GetConfession(ctx, confession.ID)
	if err != nil {
		t.Fatalf("GetConfession failed: %v", err)
	}
	if fetched == nil || fetched.Content != confession.Content {
		t.Error("Retrieved confession doesn't match")
	}
}

func TestConfessionServiceRateLimit(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.Config{MaxConfessionsPerHour: 2}
	confessionSvc := NewConfessionService(db, cfg)
	ctx := context.Background()
	userID := int64(10002)

	createUserForTest(t, db, userID, "Laki-laki", "Teknik Mesin", 2022)

	_, err := confessionSvc.CreateConfession(ctx, userID, "Confession pertama untuk testing rate limit.")
	if err != nil {
		t.Fatalf("1st confession failed: %v", err)
	}

	_, err = confessionSvc.CreateConfession(ctx, userID, "Confession kedua untuk testing rate limit.")
	if err != nil {
		t.Fatalf("2nd confession failed: %v", err)
	}

	_, err = confessionSvc.CreateConfession(ctx, userID, "Confession ketiga harus ditolak karena rate limit.")
	if err == nil {
		t.Fatal("Expected rate limit error on 3rd confession")
	}
}

func TestConfessionServiceGetLatest(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.Config{MaxConfessionsPerHour: 10}
	confessionSvc := NewConfessionService(db, cfg)
	ctx := context.Background()
	userID := int64(10003)

	createUserForTest(t, db, userID, "Perempuan", "Akuntansi", 2023)

	for i := 0; i < 5; i++ {
		_, err := confessionSvc.CreateConfession(ctx, userID, "Confession test content yang panjang untuk testing.")
		if err != nil {
			t.Fatalf("Create confession %d failed: %v", i, err)
		}
	}

	confessions, err := confessionSvc.GetLatestConfessions(ctx, 3)
	if err != nil {
		t.Fatalf("GetLatestConfessions failed: %v", err)
	}
	if len(confessions) != 3 {
		t.Fatalf("Expected 3 confessions, got %d", len(confessions))
	}
}

func TestConfessionServiceReaction(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.Config{MaxConfessionsPerHour: 10}
	confessionSvc := NewConfessionService(db, cfg)
	ctx := context.Background()

	authorID := int64(10004)
	reactorID := int64(10005)
	createUserForTest(t, db, authorID, "Laki-laki", "Teknik Sipil", 2022)
	createUserForTest(t, db, reactorID, "Perempuan", "Teknik Sipil", 2022)

	confession, _ := confessionSvc.CreateConfession(ctx, authorID, "Confession yang akan direaction.")
	if confession == nil {
		t.Fatal("Expected non-nil confession")
	}

	err := confessionSvc.ReactToConfession(ctx, confession.ID, reactorID, "❤️")
	if err != nil {
		t.Fatalf("ReactToConfession failed: %v", err)
	}

	counts, err := confessionSvc.GetReactionCounts(ctx, confession.ID)
	if err != nil {
		t.Fatalf("GetReactionCounts failed: %v", err)
	}
	if counts["❤️"] != 1 {
		t.Errorf("Expected 1 heart reaction, got %d", counts["❤️"])
	}
}

func TestConfessionServiceReactionReplace(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.Config{MaxConfessionsPerHour: 10}
	confessionSvc := NewConfessionService(db, cfg)
	ctx := context.Background()

	authorID := int64(10006)
	reactorID := int64(10007)
	createUserForTest(t, db, authorID, "Laki-laki", "Teknik Elektro", 2022)
	createUserForTest(t, db, reactorID, "Perempuan", "Teknik Elektro", 2022)

	confession, _ := confessionSvc.CreateConfession(ctx, authorID, "Testing replace reaction agar hasilnya benar.")

	_ = confessionSvc.ReactToConfession(ctx, confession.ID, reactorID, "❤️")
	_ = confessionSvc.ReactToConfession(ctx, confession.ID, reactorID, "😂")

	counts, _ := confessionSvc.GetReactionCounts(ctx, confession.ID)
	total := 0
	for _, c := range counts {
		total += c
	}
	if total != 1 {
		t.Errorf("Expected 1 total reaction (replace), got %d", total)
	}
}

func TestConfessionServiceUserNotFound(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.Config{MaxConfessionsPerHour: 3}
	confessionSvc := NewConfessionService(db, cfg)
	ctx := context.Background()

	_, err := confessionSvc.CreateConfession(ctx, 99999, "User yang tidak ada.")
	if err == nil {
		t.Error("Expected error for non-existent user")
	}
}
