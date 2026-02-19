package database

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pnj-anonymous-bot/internal/config"
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

func setupTestDB(t *testing.T) *DB {
	t.Helper()
	cfg := &config.Config{
		DBType: "sqlite",
		DBPath: filepath.Join(t.TempDir(), "test.db"),
	}
	db, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestCreateAndGetUser(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	userID := int64(1001)

	user, err := db.CreateUser(ctx, userID)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if user == nil || user.TelegramID != userID {
		t.Fatal("Created user doesn't match expected ID")
	}

	fetched, err := db.GetUser(ctx, userID)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if fetched == nil || fetched.TelegramID != userID {
		t.Fatal("Fetched user doesn't match")
	}
}

func TestGetUserNonExistent(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	user, err := db.GetUser(ctx, 99999)
	if err != nil {
		t.Fatalf("GetUser for non-existent should not error: %v", err)
	}
	if user != nil {
		t.Fatal("Expected nil user for non-existent ID")
	}
}

func TestUpdateUserFields(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	userID := int64(1002)
	_, _ = db.CreateUser(ctx, userID)

	if err := db.UpdateUserEmail(ctx, userID, "test@mhsw.pnj.ac.id"); err != nil {
		t.Fatalf("UpdateUserEmail failed: %v", err)
	}
	if err := db.UpdateUserGender(ctx, userID, "Laki-laki"); err != nil {
		t.Fatalf("UpdateUserGender failed: %v", err)
	}
	if err := db.UpdateUserDepartment(ctx, userID, "Teknik Informatika & Komputer"); err != nil {
		t.Fatalf("UpdateUserDepartment failed: %v", err)
	}
	if err := db.UpdateUserYear(ctx, userID, 2022); err != nil {
		t.Fatalf("UpdateUserYear failed: %v", err)
	}
	if err := db.UpdateUserDisplayName(ctx, userID, "TestUser123"); err != nil {
		t.Fatalf("UpdateUserDisplayName failed: %v", err)
	}
	if err := db.UpdateUserVerified(ctx, userID, true); err != nil {
		t.Fatalf("UpdateUserVerified failed: %v", err)
	}

	user, _ := db.GetUser(ctx, userID)
	if user.Email != "test@mhsw.pnj.ac.id" {
		t.Errorf("expected email test@mhsw.pnj.ac.id, got %s", user.Email)
	}
	if string(user.Gender) != "Laki-laki" {
		t.Errorf("expected gender Laki-laki, got %s", user.Gender)
	}
	if string(user.Department) != "Teknik Informatika & Komputer" {
		t.Errorf("expected dept TIK, got %s", user.Department)
	}
	if user.Year != 2022 {
		t.Errorf("expected year 2022, got %d", user.Year)
	}
	if user.DisplayName != "TestUser123" {
		t.Errorf("expected display name TestUser123, got %s", user.DisplayName)
	}
	if !user.IsVerified {
		t.Error("expected user to be verified")
	}
}

func TestUserState(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	userID := int64(1003)
	_, _ = db.CreateUser(ctx, userID)

	err := db.SetUserState(ctx, userID, models.StateAwaitingEmail, "test@mhsw.pnj.ac.id")
	if err != nil {
		t.Fatalf("SetUserState failed: %v", err)
	}

	state, data, err := db.GetUserState(ctx, userID)
	if err != nil {
		t.Fatalf("GetUserState failed: %v", err)
	}
	if state != models.StateAwaitingEmail {
		t.Errorf("expected state %s, got %s", models.StateAwaitingEmail, state)
	}
	if data != "test@mhsw.pnj.ac.id" {
		t.Errorf("expected state data 'test@mhsw.pnj.ac.id', got '%s'", data)
	}
}

func TestUserStateNonExistent(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	state, data, err := db.GetUserState(ctx, 99999)
	if err != nil {
		t.Fatalf("GetUserState for non-existent user should not error: %v", err)
	}
	if state != models.StateNone {
		t.Errorf("expected StateNone, got %s", state)
	}
	if data != "" {
		t.Errorf("expected empty data, got '%s'", data)
	}
}

func TestBanUser(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	userID := int64(1004)
	_, _ = db.CreateUser(ctx, userID)

	_ = db.UpdateUserBanned(ctx, userID, true)

	user, _ := db.GetUser(ctx, userID)
	if !user.IsBanned {
		t.Error("expected user to be banned")
	}

	_ = db.UpdateUserBanned(ctx, userID, false)
	user, _ = db.GetUser(ctx, userID)
	if user.IsBanned {
		t.Error("expected user to be unbanned")
	}
}

func TestKarmaIncrement(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	userID := int64(1005)
	_, _ = db.CreateUser(ctx, userID)

	_ = db.IncrementUserKarma(ctx, userID, 5)
	_ = db.IncrementUserKarma(ctx, userID, 3)

	user, _ := db.GetUser(ctx, userID)
	if user.Karma != 8 {
		t.Errorf("expected karma 8, got %d", user.Karma)
	}
}

func TestReportCount(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	userID := int64(1006)
	_, _ = db.CreateUser(ctx, userID)

	count, err := db.IncrementReportCount(ctx, userID)
	if err != nil {
		t.Fatalf("IncrementReportCount failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected report count 1, got %d", count)
	}

	count, _ = db.IncrementReportCount(ctx, userID)
	if count != 2 {
		t.Errorf("expected report count 2, got %d", count)
	}
}

func TestChatSessionCRUD(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	user1 := int64(2001)
	user2 := int64(2002)
	_, _ = db.CreateUser(ctx, user1)
	_, _ = db.CreateUser(ctx, user2)

	session, err := db.CreateChatSession(ctx, user1, user2)
	if err != nil {
		t.Fatalf("CreateChatSession failed: %v", err)
	}
	if !session.IsActive {
		t.Error("expected session to be active")
	}

	active, err := db.GetActiveSession(ctx, user1)
	if err != nil {
		t.Fatalf("GetActiveSession failed: %v", err)
	}
	if active == nil || active.ID != session.ID {
		t.Error("active session doesn't match")
	}

	partner, err := db.GetChatPartner(ctx, user1)
	if err != nil {
		t.Fatalf("GetChatPartner failed: %v", err)
	}
	if partner != user2 {
		t.Errorf("expected partner %d, got %d", user2, partner)
	}

	partnerReverse, _ := db.GetChatPartner(ctx, user2)
	if partnerReverse != user1 {
		t.Errorf("expected partner %d, got %d", user1, partnerReverse)
	}
}

func TestStopChat(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	user1 := int64(2003)
	user2 := int64(2004)
	_, _ = db.CreateUser(ctx, user1)
	_, _ = db.CreateUser(ctx, user2)
	_, _ = db.CreateChatSession(ctx, user1, user2)

	partnerID, err := db.StopChat(ctx, user1)
	if err != nil {
		t.Fatalf("StopChat failed: %v", err)
	}
	if partnerID != user2 {
		t.Errorf("expected partner %d, got %d", user2, partnerID)
	}

	active, _ := db.GetActiveSession(ctx, user1)
	if active != nil {
		t.Error("expected no active session after stop")
	}
}

func TestStopChatNoActiveSession(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	user1 := int64(2005)
	_, _ = db.CreateUser(ctx, user1)

	partnerID, err := db.StopChat(ctx, user1)
	if err != nil {
		t.Fatalf("StopChat with no session should not error: %v", err)
	}
	if partnerID != 0 {
		t.Errorf("expected 0 partnerID, got %d", partnerID)
	}
}

func TestConfessionCRUD(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	userID := int64(3001)
	_, _ = db.CreateUser(ctx, userID)

	confession, err := db.CreateConfession(ctx, userID, "Test confession content", "Teknik Informatika & Komputer")
	if err != nil {
		t.Fatalf("CreateConfession failed: %v", err)
	}
	if confession.AuthorID != userID {
		t.Errorf("expected author %d, got %d", userID, confession.AuthorID)
	}

	fetched, err := db.GetConfession(ctx, confession.ID)
	if err != nil {
		t.Fatalf("GetConfession failed: %v", err)
	}
	if fetched == nil || fetched.Content != "Test confession content" {
		t.Error("fetched confession doesn't match")
	}
}

func TestConfessionReactions(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	authorID := int64(3002)
	reactorID := int64(3003)
	_, _ = db.CreateUser(ctx, authorID)
	_, _ = db.CreateUser(ctx, reactorID)

	confession, _ := db.CreateConfession(ctx, authorID, "Reaction test confession", "TIK")

	err := db.AddConfessionReaction(ctx, confession.ID, reactorID, "❤️")
	if err != nil {
		t.Fatalf("AddConfessionReaction failed: %v", err)
	}

	hasReacted, _ := db.HasReacted(ctx, confession.ID, reactorID)
	if !hasReacted {
		t.Error("expected user to have reacted")
	}

	hasReacted, _ = db.HasReacted(ctx, confession.ID, authorID)
	if hasReacted {
		t.Error("author should not have reacted")
	}

	counts, err := db.GetConfessionReactionCounts(ctx, confession.ID)
	if err != nil {
		t.Fatalf("GetConfessionReactionCounts failed: %v", err)
	}
	if counts["❤️"] != 1 {
		t.Errorf("expected 1 heart, got %d", counts["❤️"])
	}
}

func TestConfessionReplies(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	authorID := int64(3004)
	replierID := int64(3005)
	_, _ = db.CreateUser(ctx, authorID)
	_, _ = db.CreateUser(ctx, replierID)

	confession, _ := db.CreateConfession(ctx, authorID, "Test with replies", "AKT")

	err := db.CreateConfessionReply(ctx, confession.ID, replierID, "Nice confession!")
	if err != nil {
		t.Fatalf("CreateConfessionReply failed: %v", err)
	}

	replies, err := db.GetConfessionReplies(ctx, confession.ID)
	if err != nil {
		t.Fatalf("GetConfessionReplies failed: %v", err)
	}
	if len(replies) != 1 {
		t.Fatalf("expected 1 reply, got %d", len(replies))
	}
	if replies[0].Content != "Nice confession!" {
		t.Errorf("expected reply content 'Nice confession!', got '%s'", replies[0].Content)
	}

	count, _ := db.GetConfessionReplyCount(ctx, confession.ID)
	if count != 1 {
		t.Errorf("expected reply count 1, got %d", count)
	}
}

func TestReportAndBlock(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	reporter := int64(4001)
	reported := int64(4002)
	_, _ = db.CreateUser(ctx, reporter)
	_, _ = db.CreateUser(ctx, reported)

	err := db.CreateReport(ctx, reporter, reported, "spam", "evidence data", 0)
	if err != nil {
		t.Fatalf("CreateReport failed: %v", err)
	}

	count, err := db.GetUserReportCount(ctx, reporter, time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("GetUserReportCount failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 report, got %d", count)
	}
}

func TestBlockUser(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	user1 := int64(4003)
	user2 := int64(4004)
	_, _ = db.CreateUser(ctx, user1)
	_, _ = db.CreateUser(ctx, user2)

	err := db.BlockUser(ctx, user1, user2)
	if err != nil {
		t.Fatalf("BlockUser failed: %v", err)
	}

	isBlocked, _ := db.IsBlocked(ctx, user1, user2)
	if !isBlocked {
		t.Error("expected user to be blocked")
	}

	isBlocked, _ = db.IsBlocked(ctx, user2, user1)
	if !isBlocked {
		t.Error("expected block to be bidirectional")
	}

	isBlocked, _ = db.IsBlocked(ctx, user1, 99999)
	if isBlocked {
		t.Error("expected non-blocked pair to not be blocked")
	}
}

func TestVerificationCode(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	userID := int64(5001)
	_, _ = db.CreateUser(ctx, userID)

	expiresAt := time.Now().Add(10 * time.Minute)
	err := db.SaveVerificationCode(ctx, userID, "test@mhsw.pnj.ac.id", "123456", expiresAt)
	if err != nil {
		t.Fatalf("SaveVerificationCode failed: %v", err)
	}

	email, valid, err := db.VerifyCode(ctx, userID, "123456")
	if err != nil {
		t.Fatalf("VerifyCode failed: %v", err)
	}
	if !valid {
		t.Error("expected code to be valid")
	}
	if email != "test@mhsw.pnj.ac.id" {
		t.Errorf("expected email test@mhsw.pnj.ac.id, got %s", email)
	}

	_, valid, _ = db.VerifyCode(ctx, userID, "123456")
	if valid {
		t.Error("expected code to be invalid after use")
	}
}

func TestVerificationCodeWrong(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	userID := int64(5002)
	_, _ = db.CreateUser(ctx, userID)

	expiresAt := time.Now().Add(10 * time.Minute)
	_ = db.SaveVerificationCode(ctx, userID, "test@mhsw.pnj.ac.id", "123456", expiresAt)

	_, valid, err := db.VerifyCode(ctx, userID, "000000")
	if err != nil {
		t.Fatalf("VerifyCode with wrong code should not error: %v", err)
	}
	if valid {
		t.Error("expected wrong code to be invalid")
	}
}

func TestGamification(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	userID := int64(6001)
	_, _ = db.CreateUser(ctx, userID)

	newLevel, leveledUp, err := db.AddPointsAndExp(ctx, userID, 10, 50)
	if err != nil {
		t.Fatalf("AddPointsAndExp failed: %v", err)
	}
	if newLevel < 1 {
		t.Errorf("expected level >= 1, got %d", newLevel)
	}

	user, _ := db.GetUser(ctx, userID)
	if user.Points != 10 {
		t.Errorf("expected 10 points, got %d", user.Points)
	}
	if user.Exp != 50 {
		t.Errorf("expected 50 exp, got %d", user.Exp)
	}

	_ = leveledUp
}

func TestDailyStreak(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	userID := int64(6002)
	_, _ = db.CreateUser(ctx, userID)

	streak, bonus, err := db.UpdateDailyStreak(ctx, userID)
	if err != nil {
		t.Fatalf("UpdateDailyStreak failed: %v", err)
	}
	if streak != 1 {
		t.Errorf("expected streak 1, got %d", streak)
	}
	if bonus {
		t.Error("first streak should not have bonus")
	}

	streak, _, _ = db.UpdateDailyStreak(ctx, userID)
	if streak != 1 {
		t.Errorf("same day streak should stay 1, got %d", streak)
	}
}

func TestLeaderboard(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	for i := int64(7001); i <= 7005; i++ {
		_, _ = db.CreateUser(ctx, i)
		_ = db.UpdateUserVerified(ctx, i, true)
		_ = db.UpdateUserDisplayName(ctx, i, "User"+string(rune('A'+i-7001)))
		_, _, _ = db.AddPointsAndExp(ctx, i, int(i-7000)*10, int(i-7000)*50)
	}

	leaderboard, err := db.GetLeaderboard(ctx, 3)
	if err != nil {
		t.Fatalf("GetLeaderboard failed: %v", err)
	}
	if len(leaderboard) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(leaderboard))
	}

	if leaderboard[0].Points < leaderboard[1].Points {
		t.Error("leaderboard should be sorted by points desc")
	}
}

func TestRoomCRUD(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	room, err := db.CreateRoom(ctx, "test-room", "Test Room", "A test room")
	if err != nil {
		t.Fatalf("CreateRoom failed: %v", err)
	}
	if room == nil || room.Slug != "test-room" {
		t.Fatal("created room doesn't match")
	}

	fetched, err := db.GetRoomBySlug(ctx, "test-room")
	if err != nil {
		t.Fatalf("GetRoomBySlug failed: %v", err)
	}
	if fetched == nil || fetched.Name != "Test Room" {
		t.Error("fetched room doesn't match")
	}

	fetchedByID, _ := db.GetRoomByID(ctx, room.ID)
	if fetchedByID == nil {
		t.Error("GetRoomByID should return a room")
	}
}

func TestRoomMembers(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	user1 := int64(8001)
	user2 := int64(8002)
	_, _ = db.CreateUser(ctx, user1)
	_, _ = db.CreateUser(ctx, user2)

	room, _ := db.CreateRoom(ctx, "member-test", "Member Test Room", "Testing room members")

	_ = db.AddRoomMember(ctx, room.ID, user1)
	_ = db.AddRoomMember(ctx, room.ID, user2)

	members, err := db.GetRoomMembers(ctx, room.ID)
	if err != nil {
		t.Fatalf("GetRoomMembers failed: %v", err)
	}
	if len(members) != 2 {
		t.Errorf("expected 2 members, got %d", len(members))
	}

	userRoom, _ := db.GetUserRoom(ctx, user1)
	if userRoom == nil || userRoom.Slug != "member-test" {
		t.Error("GetUserRoom should return the room user is in")
	}

	_ = db.RemoveRoomMember(ctx, room.ID, user1)
	members, _ = db.GetRoomMembers(ctx, room.ID)
	if len(members) != 1 {
		t.Errorf("expected 1 member after remove, got %d", len(members))
	}
}

func TestRemoveMemberFromAllRooms(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	userID := int64(8003)
	_, _ = db.CreateUser(ctx, userID)

	room1, _ := db.CreateRoom(ctx, "room-a", "Room A", "First test room")
	room2, _ := db.CreateRoom(ctx, "room-b", "Room B", "Second test room")
	_ = db.AddRoomMember(ctx, room1.ID, userID)
	_ = db.AddRoomMember(ctx, room2.ID, userID)

	_ = db.RemoveMemberFromAllRooms(ctx, userID)

	members1, _ := db.GetRoomMembers(ctx, room1.ID)
	members2, _ := db.GetRoomMembers(ctx, room2.ID)
	if len(members1) != 0 || len(members2) != 0 {
		t.Error("user should be removed from all rooms")
	}
}

func TestGetOnlineUserCount(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	for i := int64(9001); i <= 9003; i++ {
		_, _ = db.CreateUser(ctx, i)
		_ = db.UpdateUserVerified(ctx, i, true)
	}

	count, err := db.GetOnlineUserCount(ctx)
	if err != nil {
		t.Fatalf("GetOnlineUserCount failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 online users, got %d", count)
	}
}

func TestGetDepartmentUserCount(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	for i := int64(9004); i <= 9006; i++ {
		_, _ = db.CreateUser(ctx, i)
		_ = db.UpdateUserVerified(ctx, i, true)
		_ = db.UpdateUserDepartment(ctx, i, "Teknik Informatika & Komputer")
	}
	_, _ = db.CreateUser(ctx, 9007)
	_ = db.UpdateUserVerified(ctx, 9007, true)
	_ = db.UpdateUserDepartment(ctx, 9007, "Teknik Mesin")

	count, _ := db.GetDepartmentUserCount(ctx, "Teknik Informatika & Komputer")
	if count != 3 {
		t.Errorf("expected 3 TIK users, got %d", count)
	}
}

func TestGetAllVerifiedUsers(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	for i := int64(9008); i <= 9010; i++ {
		_, _ = db.CreateUser(ctx, i)
		_ = db.UpdateUserVerified(ctx, i, true)
	}

	_, _ = db.CreateUser(ctx, 9011)
	_ = db.UpdateUserVerified(ctx, 9011, true)
	_ = db.UpdateUserBanned(ctx, 9011, true)

	users, err := db.GetAllVerifiedUsers(ctx)
	if err != nil {
		t.Fatalf("GetAllVerifiedUsers failed: %v", err)
	}
	if len(users) != 3 {
		t.Errorf("expected 3 verified non-banned users, got %d", len(users))
	}
}

func TestIsUserProfileComplete(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	userID := int64(9012)
	_, _ = db.CreateUser(ctx, userID)

	complete, _ := db.IsUserProfileComplete(ctx, userID)
	if complete {
		t.Error("incomplete profile should return false")
	}

	_ = db.UpdateUserVerified(ctx, userID, true)
	_ = db.UpdateUserGender(ctx, userID, "Laki-laki")
	_ = db.UpdateUserDepartment(ctx, userID, "Teknik Informatika & Komputer")
	_ = db.UpdateUserYear(ctx, userID, 2022)

	complete, _ = db.IsUserProfileComplete(ctx, userID)
	if !complete {
		t.Error("complete profile should return true")
	}
}

func TestWhisper(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	senderID := int64(9013)
	_, _ = db.CreateUser(ctx, senderID)

	id, err := db.CreateWhisper(ctx, senderID, "Teknik Mesin", "Hello whisper", "TIK", "Laki-laki")
	if err != nil {
		t.Fatalf("CreateWhisper failed: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero whisper ID")
	}
}

func TestPing(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	err := db.PingContext(ctx)
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

func TestTotalChatSessions(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	user1 := int64(2010)
	user2 := int64(2011)
	user3 := int64(2012)
	_, _ = db.CreateUser(ctx, user1)
	_, _ = db.CreateUser(ctx, user2)
	_, _ = db.CreateUser(ctx, user3)

	_, _ = db.CreateChatSession(ctx, user1, user2)
	_, _ = db.CreateChatSession(ctx, user1, user3)

	count, err := db.GetTotalChatSessions(ctx, user1)
	if err != nil {
		t.Fatalf("GetTotalChatSessions failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 chat sessions, got %d", count)
	}
}
