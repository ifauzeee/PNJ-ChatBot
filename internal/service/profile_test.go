package service

import (
	"context"
	"testing"

	"github.com/pnj-anonymous-bot/internal/config"
	"github.com/pnj-anonymous-bot/internal/models"
)

func TestProfileServiceSetGenderValid(t *testing.T) {
	db := setupTestDB(t)
	profileSvc := NewProfileService(db, &config.Config{})
	ctx := context.Background()
	userID := int64(11001)
	createUserForTest(t, db, userID, "", "", 0)

	err := profileSvc.SetGender(ctx, userID, string(models.GenderMale))
	if err != nil {
		t.Fatalf("SetGender with valid gender failed: %v", err)
	}

	state, _, err := db.GetUserState(ctx, userID)
	if err != nil {
		t.Fatalf("failed to get state: %v", err)
	}
	if state != models.StateAwaitingYear {
		t.Fatalf("expected state awaiting_year, got %s", state)
	}
}

func TestProfileServiceSetGenderInvalid(t *testing.T) {
	db := setupTestDB(t)
	profileSvc := NewProfileService(db, &config.Config{})
	ctx := context.Background()
	userID := int64(11002)
	createUserForTest(t, db, userID, "", "", 0)

	err := profileSvc.SetGender(ctx, userID, "InvalidGender")
	if err == nil {
		t.Fatal("expected error for invalid gender")
	}
}

func TestProfileServiceSetDepartmentValid(t *testing.T) {
	db := setupTestDB(t)
	profileSvc := NewProfileService(db, &config.Config{})
	ctx := context.Background()
	userID := int64(11003)
	createUserForTest(t, db, userID, string(models.GenderFemale), "", 2022)

	err := profileSvc.SetDepartment(ctx, userID, string(models.DeptTeknikInformatika))
	if err != nil {
		t.Fatalf("SetDepartment failed: %v", err)
	}

	user, err := db.GetUser(ctx, userID)
	if err != nil || user == nil {
		t.Fatal("failed to get user after setting department")
	}
	if user.DisplayName == "" {
		t.Error("expected display name to be generated")
	}
	if string(user.Department) != string(models.DeptTeknikInformatika) {
		t.Errorf("expected department %s, got %s", models.DeptTeknikInformatika, user.Department)
	}
}

func TestProfileServiceSetDepartmentInvalid(t *testing.T) {
	db := setupTestDB(t)
	profileSvc := NewProfileService(db, &config.Config{})
	ctx := context.Background()
	userID := int64(11004)
	createUserForTest(t, db, userID, "", "", 0)

	err := profileSvc.SetDepartment(ctx, userID, "FakeJurusan")
	if err == nil {
		t.Fatal("expected error for invalid department")
	}
}

func TestProfileServiceSetYearBoundary(t *testing.T) {
	db := setupTestDB(t)
	profileSvc := NewProfileService(db, &config.Config{})
	ctx := context.Background()
	userID := int64(11005)
	createUserForTest(t, db, userID, "", "", 0)

	currentYear := models.CurrentEntryYear()
	err := profileSvc.SetYear(ctx, userID, currentYear)
	if err != nil {
		t.Fatalf("SetYear with current year failed: %v", err)
	}

	err = profileSvc.SetYear(ctx, userID, currentYear+1)
	if err == nil {
		t.Fatal("expected error for future year")
	}

	err = profileSvc.SetYear(ctx, userID, models.MinEntryYear-1)
	if err == nil {
		t.Fatal("expected error for too old year")
	}
}

func TestProfileServiceUpdateGender(t *testing.T) {
	db := setupTestDB(t)
	profileSvc := NewProfileService(db, &config.Config{})
	ctx := context.Background()
	userID := int64(11006)
	createUserForTest(t, db, userID, string(models.GenderMale), "", 0)

	err := profileSvc.UpdateGender(ctx, userID, string(models.GenderFemale))
	if err != nil {
		t.Fatalf("UpdateGender failed: %v", err)
	}

	user, _ := db.GetUser(ctx, userID)
	if string(user.Gender) != string(models.GenderFemale) {
		t.Errorf("expected Perempuan, got %s", user.Gender)
	}
}

func TestProfileServiceUpdateYear(t *testing.T) {
	db := setupTestDB(t)
	profileSvc := NewProfileService(db, &config.Config{})
	ctx := context.Background()
	userID := int64(11007)
	createUserForTest(t, db, userID, "", "", 2022)

	currentYear := models.CurrentEntryYear()
	err := profileSvc.UpdateYear(ctx, userID, currentYear)
	if err != nil {
		t.Fatalf("UpdateYear failed: %v", err)
	}

	user, _ := db.GetUser(ctx, userID)
	if user.Year != currentYear {
		t.Errorf("expected year %d, got %d", currentYear, user.Year)
	}
}

func TestProfileServiceUpdateDepartment(t *testing.T) {
	db := setupTestDB(t)
	profileSvc := NewProfileService(db, &config.Config{})
	ctx := context.Background()
	userID := int64(11008)
	createUserForTest(t, db, userID, "", "Teknik Mesin", 0)

	err := profileSvc.UpdateDepartment(ctx, userID, string(models.DeptAkuntansi))
	if err != nil {
		t.Fatalf("UpdateDepartment failed: %v", err)
	}

	user, _ := db.GetUser(ctx, userID)
	if string(user.Department) != string(models.DeptAkuntansi) {
		t.Errorf("expected Akuntansi, got %s", user.Department)
	}
}

func TestProfileServiceReportUser(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.Config{MaxReportsPerDay: 3, AutoBanReportCount: 5}
	profileSvc := NewProfileService(db, cfg)
	ctx := context.Background()

	reporterID := int64(11009)
	reportedID := int64(11010)
	createUserForTest(t, db, reporterID, "Laki-laki", "Teknik Sipil", 2022)
	createUserForTest(t, db, reportedID, "Perempuan", "Teknik Sipil", 2022)

	newCount, err := profileSvc.ReportUser(ctx, reporterID, reportedID, "spam", "", 0)
	if err != nil {
		t.Fatalf("ReportUser failed: %v", err)
	}
	if newCount != 1 {
		t.Errorf("expected report count 1, got %d", newCount)
	}
}

func TestProfileServiceReportUserRateLimit(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.Config{MaxReportsPerDay: 2, AutoBanReportCount: 10}
	profileSvc := NewProfileService(db, cfg)
	ctx := context.Background()

	reporterID := int64(11011)
	reportedID1 := int64(11012)
	reportedID2 := int64(11013)
	reportedID3 := int64(11014)
	createUserForTest(t, db, reporterID, "Laki-laki", "Akuntansi", 2022)
	createUserForTest(t, db, reportedID1, "Perempuan", "Akuntansi", 2022)
	createUserForTest(t, db, reportedID2, "Laki-laki", "Akuntansi", 2022)
	createUserForTest(t, db, reportedID3, "Perempuan", "Akuntansi", 2022)

	_, _ = profileSvc.ReportUser(ctx, reporterID, reportedID1, "reason1", "", 0)
	_, _ = profileSvc.ReportUser(ctx, reporterID, reportedID2, "reason2", "", 0)

	_, err := profileSvc.ReportUser(ctx, reporterID, reportedID3, "reason3", "", 0)
	if err == nil {
		t.Fatal("Expected rate limit error on 3rd report")
	}
}

func TestProfileServiceAutoban(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.Config{MaxReportsPerDay: 100, AutoBanReportCount: 2}
	profileSvc := NewProfileService(db, cfg)
	ctx := context.Background()

	reporter1 := int64(11015)
	reporter2 := int64(11016)
	reported := int64(11017)
	createUserForTest(t, db, reporter1, "Laki-laki", "Teknik Mesin", 2022)
	createUserForTest(t, db, reporter2, "Perempuan", "Teknik Mesin", 2022)
	createUserForTest(t, db, reported, "Laki-laki", "Teknik Mesin", 2022)

	_, _ = profileSvc.ReportUser(ctx, reporter1, reported, "reason1", "", 0)
	_, _ = profileSvc.ReportUser(ctx, reporter2, reported, "reason2", "", 0)

	user, _ := db.GetUser(ctx, reported)
	if user == nil {
		t.Fatal("expected user to exist")
	}
	if !user.IsBanned {
		t.Error("Expected user to be auto-banned after reaching report threshold")
	}
}

func TestProfileServiceBlockUser(t *testing.T) {
	db := setupTestDB(t)
	profileSvc := NewProfileService(db, &config.Config{})
	ctx := context.Background()

	user1 := int64(11018)
	user2 := int64(11019)
	createUserForTest(t, db, user1, "", "", 0)
	createUserForTest(t, db, user2, "", "", 0)

	err := profileSvc.BlockUser(ctx, user1, user2)
	if err != nil {
		t.Fatalf("BlockUser failed: %v", err)
	}

	isBlocked, err := db.IsBlocked(ctx, user1, user2)
	if err != nil {
		t.Fatalf("IsBlocked failed: %v", err)
	}
	if !isBlocked {
		t.Error("Expected user to be blocked")
	}

	isBlocked, _ = db.IsBlocked(ctx, user2, user1)
	if !isBlocked {
		t.Error("Block should be bidirectional")
	}
}

func TestProfileServiceSendWhisper(t *testing.T) {
	db := setupTestDB(t)
	profileSvc := NewProfileService(db, &config.Config{})
	ctx := context.Background()

	senderID := int64(11020)
	targetID1 := int64(11021)
	targetID2 := int64(11022)
	createUserForTest(t, db, senderID, "Laki-laki", "Teknik Informatika & Komputer", 2022)
	createUserForTest(t, db, targetID1, "Perempuan", "Teknik Mesin", 2022)
	createUserForTest(t, db, targetID2, "Laki-laki", "Teknik Mesin", 2022)
	_ = db.UpdateUserVerified(ctx, targetID1, true)
	_ = db.UpdateUserVerified(ctx, targetID2, true)

	targets, err := profileSvc.SendWhisper(ctx, senderID, "Teknik Mesin", "Hello dari TIK ke Mesin!")
	if err != nil {
		t.Fatalf("SendWhisper failed: %v", err)
	}
	if len(targets) != 2 {
		t.Errorf("Expected 2 targets, got %d", len(targets))
	}
}

func TestProfileServiceSendWhisperUserNotFound(t *testing.T) {
	db := setupTestDB(t)
	profileSvc := NewProfileService(db, &config.Config{})
	ctx := context.Background()

	_, err := profileSvc.SendWhisper(ctx, 99999, "Teknik Mesin", "User not found test message.")
	if err == nil {
		t.Error("Expected error for non-existent sender")
	}
}

func TestProfileServiceGetProfile(t *testing.T) {
	db := setupTestDB(t)
	profileSvc := NewProfileService(db, &config.Config{})
	ctx := context.Background()
	userID := int64(11023)
	createUserForTest(t, db, userID, "Laki-laki", "Akuntansi", 2022)

	user, err := profileSvc.GetProfile(ctx, userID)
	if err != nil {
		t.Fatalf("GetProfile failed: %v", err)
	}
	if user == nil {
		t.Fatal("Expected non-nil user")
	}
	if user.TelegramID != userID {
		t.Errorf("Expected telegram ID %d, got %d", userID, user.TelegramID)
	}
}

func TestProfileServiceGetStats(t *testing.T) {
	db := setupTestDB(t)
	profileSvc := NewProfileService(db, &config.Config{})
	ctx := context.Background()
	userID := int64(11024)
	createUserForTest(t, db, userID, "Perempuan", "Teknik Elektro", 2023)

	totalChats, totalConfessions, totalReactions, daysSinceJoined, err := profileSvc.GetStats(ctx, userID)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if totalChats != 0 || totalConfessions != 0 || totalReactions != 0 {
		t.Errorf("Expected all zeros for new user, got chats=%d conf=%d react=%d",
			totalChats, totalConfessions, totalReactions)
	}
	if daysSinceJoined < 0 {
		t.Errorf("daysSinceJoined should be >= 0, got %d", daysSinceJoined)
	}
}
