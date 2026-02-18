package service

import (
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
	"github.com/pnj-anonymous-bot/internal/models"
)

func TestMain(m *testing.M) {
	_ = os.Setenv("APP_ENV", "test")
	_ = os.Setenv("LOG_LEVEL", "error")
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

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func setupTestChatService(t *testing.T, maxSearchPerMinute int) (*ChatService, *database.DB) {
	t.Helper()

	db := setupTestDB(t)

	mr := miniredis.RunT(t)
	t.Setenv("REDIS_URL", mr.Addr())

	redisSvc := NewRedisService()
	t.Cleanup(func() {
		_ = redisSvc.client.Close()
		mr.Close()
	})

	return NewChatService(db, redisSvc, maxSearchPerMinute), db
}

func createUserForTest(t *testing.T, db *database.DB, id int64, gender, dept string, year int) {
	t.Helper()

	if _, err := db.CreateUser(id); err != nil {
		t.Fatalf("failed to create user %d: %v", id, err)
	}
	if gender != "" {
		if err := db.UpdateUserGender(id, gender); err != nil {
			t.Fatalf("failed to set gender for user %d: %v", id, err)
		}
	}
	if dept != "" {
		if err := db.UpdateUserDepartment(id, dept); err != nil {
			t.Fatalf("failed to set dept for user %d: %v", id, err)
		}
	}
	if year != 0 {
		if err := db.UpdateUserYear(id, year); err != nil {
			t.Fatalf("failed to set year for user %d: %v", id, err)
		}
	}
}

func TestChatServiceSearchPartnerMatchesAndUpdatesState(t *testing.T) {
	chatSvc, db := setupTestChatService(t, 10)

	createUserForTest(t, db, 1001, string(models.GenderMale), string(models.DeptTeknikInformatika), 2022)
	createUserForTest(t, db, 1002, string(models.GenderMale), string(models.DeptTeknikInformatika), 2022)

	matchID, err := chatSvc.SearchPartner(1002, "", "", 0)
	if err != nil {
		t.Fatalf("unexpected error when enqueueing user2: %v", err)
	}
	if matchID != 0 {
		t.Fatalf("expected no immediate match for user2, got %d", matchID)
	}

	matchID, err = chatSvc.SearchPartner(1001, string(models.DeptTeknikInformatika), string(models.GenderMale), 2022)
	if err != nil {
		t.Fatalf("unexpected error when matching user1: %v", err)
	}
	if matchID != 1002 {
		t.Fatalf("expected matchID=1002, got %d", matchID)
	}

	state1, _, err := db.GetUserState(1001)
	if err != nil {
		t.Fatalf("failed to get state user1: %v", err)
	}
	state2, _, err := db.GetUserState(1002)
	if err != nil {
		t.Fatalf("failed to get state user2: %v", err)
	}
	if state1 != models.StateInChat || state2 != models.StateInChat {
		t.Fatalf("expected both users in_chat, got user1=%s user2=%s", state1, state2)
	}
}

func TestChatServiceFiltering(t *testing.T) {
	chatSvc, db := setupTestChatService(t, 10)

	createUserForTest(t, db, 5001, string(models.GenderMale), string(models.DeptTeknikInformatika), 2022)
	createUserForTest(t, db, 5002, string(models.GenderFemale), string(models.DeptTeknikMesin), 2022)
	createUserForTest(t, db, 5003, string(models.GenderMale), string(models.DeptTeknikInformatika), 2021)

	chatSvc.SearchPartner(5002, "NON_EXISTENT", "", 0)
	chatSvc.SearchPartner(5003, "NON_EXISTENT", "", 0)

	matchID, err := chatSvc.SearchPartner(5001, string(models.DeptTeknikInformatika), "", 0)
	if err != nil {
		t.Logf("SearchPartner error: %v", err)
	}
	if matchID != 5003 {
		t.Errorf("Expected match with User C (5003) due to TIK filter, got %d", matchID)
	}

	chatSvc.StopChat(5001)
	db.SetUserState(5001, models.StateNone, "")
	db.SetUserState(5002, models.StateNone, "")
	db.SetUserState(5003, models.StateNone, "")

	chatSvc.SearchPartner(5002, "NON_EXISTENT", "", 0)
	chatSvc.SearchPartner(5003, "NON_EXISTENT", "", 0)
	matchID, _ = chatSvc.SearchPartner(5001, "", "", 2022)
	if matchID != 5002 {
		t.Errorf("Expected match with User B (5002) due to Year filter, got %d", matchID)
	}
}

func TestChatServiceSearchPartnerRateLimit(t *testing.T) {
	chatSvc, db := setupTestChatService(t, 1)
	createUserForTest(t, db, 2001, "", "", 0)

	if _, err := chatSvc.SearchPartner(2001, "", "", 0); err != nil {
		t.Fatalf("first search should pass, got: %v", err)
	}

	_, err := chatSvc.SearchPartner(2001, "", "", 0)
	if err == nil {
		t.Fatalf("expected rate limit error on second search")
	}
	if !strings.Contains(err.Error(), "terlalu sering mencari partner") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestChatServiceProcessQueueTimeoutClearsFilters(t *testing.T) {
	chatSvc, _ := setupTestChatService(t, 10)

	item := QueueItem{
		TelegramID: 3001,
		Dept:       string(models.DeptAkuntansi),
		Gender:     string(models.GenderFemale),
		Year:       2021,
		JoinedAt:   time.Now().Add(-2 * time.Minute).Unix(),
	}

	if err := chatSvc.redis.AddToQueue(int64(3001), item); err != nil {
		t.Fatalf("failed to enqueue queue item: %v", err)
	}

	updatedIDs, err := chatSvc.ProcessQueueTimeout(60)
	if err != nil {
		t.Fatalf("unexpected timeout processing error: %v", err)
	}
	if !slices.Contains(updatedIDs, int64(3001)) {
		t.Fatalf("expected updated IDs to contain 3001, got %v", updatedIDs)
	}

	storedRaw, err := chatSvc.redis.client.LIndex(chatSvc.redis.ctx, "chat_queue", 0).Result()
	if err != nil {
		t.Fatalf("failed to read updated queue item: %v", err)
	}

	var updated QueueItem
	if err := json.Unmarshal([]byte(storedRaw), &updated); err != nil {
		t.Fatalf("failed to unmarshal updated queue item: %v", err)
	}

	if updated.Dept != "" || updated.Gender != "" || updated.Year != 0 {
		t.Fatalf("expected filters to be cleared, got %+v", updated)
	}
}

func TestProfileServiceOnboardingFlow(t *testing.T) {
	db := setupTestDB(t)
	profileSvc := NewProfileService(db, &config.Config{})
	userID := int64(4001)

	createUserForTest(t, db, userID, "", "", 0)

	if err := profileSvc.SetGender(userID, string(models.GenderMale)); err != nil {
		t.Fatalf("SetGender failed: %v", err)
	}
	state, _, err := db.GetUserState(userID)
	if err != nil {
		t.Fatalf("failed to get state after SetGender: %v", err)
	}
	if state != models.StateAwaitingYear {
		t.Fatalf("expected state awaiting_year, got %s", state)
	}

	currentYear := models.CurrentEntryYear()
	if err := profileSvc.SetYear(userID, currentYear); err != nil {
		t.Fatalf("SetYear failed: %v", err)
	}
	state, _, err = db.GetUserState(userID)
	if err != nil {
		t.Fatalf("failed to get state after SetYear: %v", err)
	}
	if state != models.StateAwaitingDept {
		t.Fatalf("expected state awaiting_department, got %s", state)
	}

	if err := profileSvc.SetDepartment(userID, string(models.DeptTeknikMesin)); err != nil {
		t.Fatalf("SetDepartment failed: %v", err)
	}
	state, _, err = db.GetUserState(userID)
	if err != nil {
		t.Fatalf("failed to get state after SetDepartment: %v", err)
	}
	if state != models.StateNone {
		t.Fatalf("expected state none, got %s", state)
	}

	user, err := db.GetUser(userID)
	if err != nil {
		t.Fatalf("failed to load user: %v", err)
	}
	if user == nil || user.DisplayName == "" {
		t.Fatalf("expected non-empty display name after SetDepartment")
	}
}

func TestProfileServiceRejectsInvalidYear(t *testing.T) {
	db := setupTestDB(t)
	profileSvc := NewProfileService(db, &config.Config{})
	userID := int64(5001)

	createUserForTest(t, db, userID, "", "", 0)

	invalidFutureYear := models.CurrentEntryYear() + 1
	if err := profileSvc.SetYear(userID, invalidFutureYear); err == nil {
		t.Fatalf("expected SetYear to reject year %d", invalidFutureYear)
	}

	if err := profileSvc.UpdateYear(userID, models.MinEntryYear-1); err == nil {
		t.Fatalf("expected UpdateYear to reject year %d", models.MinEntryYear-1)
	}
}
