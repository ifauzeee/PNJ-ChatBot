package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/pnj-anonymous-bot/internal/config"
	"github.com/pnj-anonymous-bot/internal/database"
	"github.com/pnj-anonymous-bot/internal/logger"
)

func TestMain(m *testing.M) {
	_ = os.Setenv("APP_ENV", "test")
	_ = os.Setenv("LOG_LEVEL", "debug")
	logger.Init()
	code := m.Run()
	_ = logger.Log.Sync()
	os.Exit(code)
}

func setupTestDB(t *testing.T) *database.DB {
	t.Helper()

	cfg := &config.Config{
		DBType: "sqlite",
		DBPath: filepath.Join(t.TempDir(), "test.db"),
	}

	db, err := database.New(cfg)
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })
	return db
}

func createUserForTest(t *testing.T, db *database.DB, telegramID int64, gender, department string, year int) {
	t.Helper()
	ctx := context.Background()
	_, err := db.CreateUser(ctx, telegramID)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	if gender != "" {
		_ = db.UpdateUserGender(ctx, telegramID, gender)
	}
	if department != "" {
		_ = db.UpdateUserDepartment(ctx, telegramID, department)
	}
	if year != 0 {
		_ = db.UpdateUserYear(ctx, telegramID, year)
	}
	_ = db.UpdateUserVerified(ctx, telegramID, true)
}

func setupTestRedis(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	_ = os.Setenv("REDIS_URL", "redis://"+mr.Addr())
	return mr
}

func TestChatServiceMatch(t *testing.T) {
	db := setupTestDB(t)
	mr := setupTestRedis(t)
	redisSvc := NewRedisService()
	chatSvc := NewChatService(db, redisSvc, 5)
	ctx := context.Background()

	user1 := int64(3001)
	user2 := int64(3002)

	createUserForTest(t, db, user1, "Laki-laki", "Teknik Informatika & Komputer", 2022)
	createUserForTest(t, db, user2, "Perempuan", "Teknik Informatika & Komputer", 2022)

	partner1, err := chatSvc.SearchPartner(ctx, user1, "", "", 0)
	if err != nil {
		t.Fatalf("SearchPartner for user1 failed: %v", err)
	}
	if partner1 != 0 {
		t.Error("User 1 should be in queue, not matched immediately")
	}

	partner2, err := chatSvc.SearchPartner(ctx, user2, "", "", 0)
	if err != nil {
		t.Fatalf("SearchPartner for user2 failed: %v", err)
	}
	if partner2 != user1 {
		t.Errorf("Expected User 2 to match with User 1, got %d", partner2)
	}

	session, _ := db.GetActiveSession(ctx, user1)
	if session == nil {
		t.Fatal("Active session not found")
	}
	if (session.User1ID == user1 && session.User2ID == user2) || (session.User1ID == user2 && session.User2ID == user1) {
	} else {
		t.Errorf("Active session incorrect: user1=%d, user2=%d, got user1=%d, user2=%d", user1, user2, session.User1ID, session.User2ID)
	}

	mr.FlushAll()
}

func TestChatServiceStop(t *testing.T) {
	db := setupTestDB(t)
	mr := setupTestRedis(t)
	redisSvc := NewRedisService()
	chatSvc := NewChatService(db, redisSvc, 5)
	ctx := context.Background()

	user1, user2 := int64(3003), int64(3004)
	createUserForTest(t, db, user1, "", "", 0)
	createUserForTest(t, db, user2, "", "", 0)

	_, _ = chatSvc.SearchPartner(ctx, user1, "", "", 0)
	_, _ = chatSvc.SearchPartner(ctx, user2, "", "", 0)

	partnerID, err := chatSvc.StopChat(ctx, user1)
	if err != nil {
		t.Fatalf("StopChat failed: %v", err)
	}
	if partnerID != user2 {
		t.Errorf("Expected partner %d, got %d", user2, partnerID)
	}

	session, _ := db.GetActiveSession(ctx, user1)
	if session != nil {
		t.Error("Session should be closed")
	}

	mr.FlushAll()
}

func TestChatServiceMatchFilters(t *testing.T) {
	db := setupTestDB(t)
	mr := setupTestRedis(t)
	redisSvc := NewRedisService()
	chatSvc := NewChatService(db, redisSvc, 5)
	ctx := context.Background()

	user1, user2, user3 := int64(3005), int64(3006), int64(3007)
	createUserForTest(t, db, user1, "Perempuan", "Teknik Informatika & Komputer", 2022)
	createUserForTest(t, db, user2, "Laki-laki", "Teknik Mesin", 2022)
	createUserForTest(t, db, user3, "Laki-laki", "Teknik Informatika & Komputer", 2023)

	_, _ = chatSvc.SearchPartner(ctx, user1, "", "", 0)
	_, _ = chatSvc.SearchPartner(ctx, user2, "Teknik Sipil", "", 0)

	partner3, err := chatSvc.SearchPartner(ctx, user3, "Teknik Informatika & Komputer", "", 0)
	if err != nil {
		t.Fatalf("SearchPartner failed: %v", err)
	}
	if partner3 != user1 {
		t.Errorf("Expected User 3 to match with User 1 (TIK), got %d", partner3)
	}

	mr.FlushAll()
}

func TestChatServiceQueueTimeout(t *testing.T) {
	db := setupTestDB(t)
	mr := setupTestRedis(t)
	redisSvc := NewRedisService()
	chatSvc := NewChatService(db, redisSvc, 5)
	ctx := context.Background()

	userID := int64(3008)
	createUserForTest(t, db, userID, "", "", 0)

	_, err := chatSvc.SearchPartner(ctx, userID, "TIK", "L", 0)
	if err != nil {
		t.Fatalf("SearchPartner failed: %v", err)
	}

	queueKey := "chat_queue"
	list, _ := mr.List(queueKey)
	if len(list) == 0 {
		t.Fatal("Queue is empty")
	}

	var item QueueItem
	_ = json.Unmarshal([]byte(list[0]), &item)
	item.JoinedAt = time.Now().Add(-10 * time.Minute).Unix()

	if err := redisSvc.AddToQueue(ctx, userID, item); err != nil {
		t.Fatalf("Failed to update queue item for test: %v", err)
	}

	timedOut, err := chatSvc.ProcessQueueTimeout(ctx, 300)
	if err != nil {
		t.Fatalf("ProcessQueueTimeout failed: %v", err)
	}

	if !slices.Contains(timedOut, userID) {
		t.Error("User should be in timed out list")
	}

	listAfter, _ := mr.List(queueKey)
	if len(listAfter) == 0 {
		t.Fatal("User was incorrectly removed from queue")
	}
	var itemAfter QueueItem
	_ = json.Unmarshal([]byte(listAfter[0]), &itemAfter)
	if itemAfter.Dept != "" || itemAfter.Gender != "" {
		t.Errorf("Filters were not relaxed: dept=%s, gender=%s", itemAfter.Dept, itemAfter.Gender)
	}

	mr.FlushAll()
}

func TestEvidenceService(t *testing.T) {
	db := setupTestDB(t)
	mr := setupTestRedis(t)
	redisClient := NewRedisService().GetClient()
	evidenceSvc := NewEvidenceService(db, redisClient)
	ctx := context.Background()

	sessionID := int64(501)
	evidenceSvc.LogMessage(ctx, sessionID, 101, "Hello", "text")
	evidenceSvc.LogMessage(ctx, sessionID, 102, "Hi there", "text")

	evidence, err := evidenceSvc.GetEvidence(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetEvidence failed: %v", err)
	}

	if !strings.Contains(evidence, "Hello") || !strings.Contains(evidence, "Hi there") {
		t.Error("Evidence content missing")
	}

	evidenceSvc.ClearEvidence(ctx, sessionID)
	val := mr.Exists("chat_evidence:501")
	if val {
		t.Error("Evidence should be cleared in Redis")
	}
}

func TestEvidenceServiceCap(t *testing.T) {
	db := setupTestDB(t)
	_ = setupTestRedis(t)
	redisClient := NewRedisService().GetClient()
	evidenceSvc := NewEvidenceService(db, redisClient)
	ctx := context.Background()

	sessionID := int64(502)
	for i := 0; i < 30; i++ {
		evidenceSvc.LogMessage(ctx, sessionID, 101, strings.Repeat("a", i), "text")
	}

	evidence, _ := evidenceSvc.GetEvidence(ctx, sessionID)
	lines := strings.Split(strings.TrimSpace(evidence), "\n")
	if len(lines) > 20 {
		t.Errorf("Expected max 20 lines, got %d", len(lines))
	}
}
