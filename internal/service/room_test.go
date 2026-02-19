package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/pnj-anonymous-bot/internal/config"
	"github.com/pnj-anonymous-bot/internal/models"
)

func TestRoomServiceCreateRoom(t *testing.T) {
	db := setupTestDB(t)
	roomSvc := NewRoomService(db)
	ctx := context.Background()

	room, err := roomSvc.CreateRoom(ctx, "Gaming Lounge", "Room untuk gamers PNJ agar bisa ngobrol")
	if err != nil {
		t.Fatalf("CreateRoom failed: %v", err)
	}
	if room.Slug != "gaming-lounge" {
		t.Errorf("Expected slug 'gaming-lounge', got '%s'", room.Slug)
	}
	if room.Name != "Gaming Lounge" {
		t.Errorf("Expected name 'Gaming Lounge', got '%s'", room.Name)
	}
}

func TestRoomServiceCreateDuplicateRoom(t *testing.T) {
	db := setupTestDB(t)
	roomSvc := NewRoomService(db)
	ctx := context.Background()

	_, _ = roomSvc.CreateRoom(ctx, "Gaming Lounge", "First room created for testing purposes")
	_, err := roomSvc.CreateRoom(ctx, "Gaming Lounge", "Second with same name should fail")
	if err == nil {
		t.Fatal("Expected error for duplicate room name")
	}
}

func TestRoomServiceCreateRoomInvalidName(t *testing.T) {
	db := setupTestDB(t)
	roomSvc := NewRoomService(db)
	ctx := context.Background()

	_, err := roomSvc.CreateRoom(ctx, "!!!", "Testing invalid room name with special chars")
	if err == nil {
		t.Fatal("Expected error for invalid room name")
	}
}

func TestRoomServiceJoinAndLeave(t *testing.T) {
	db := setupTestDB(t)
	roomSvc := NewRoomService(db)
	ctx := context.Background()
	userID := int64(12001)
	createUserForTest(t, db, userID, "Laki-laki", "Teknik Informatika & Komputer", 2022)

	room, _ := roomSvc.CreateRoom(ctx, "Coding Club", "Room untuk coding enthusiasts di PNJ")

	joinedRoom, err := roomSvc.JoinRoom(ctx, userID, room.Slug)
	if err != nil {
		t.Fatalf("JoinRoom failed: %v", err)
	}
	if joinedRoom.Slug != room.Slug {
		t.Errorf("Expected slug '%s', got '%s'", room.Slug, joinedRoom.Slug)
	}

	state, stateData, _ := db.GetUserState(ctx, userID)
	if state != models.StateInCircle {
		t.Errorf("Expected state in_circle, got %s", state)
	}
	if stateData != room.Slug {
		t.Errorf("Expected state data '%s', got '%s'", room.Slug, stateData)
	}

	err = roomSvc.LeaveRoom(ctx, userID)
	if err != nil {
		t.Fatalf("LeaveRoom failed: %v", err)
	}

	state, _, _ = db.GetUserState(ctx, userID)
	if state != models.StateNone {
		t.Errorf("Expected state none after leaving, got %s", state)
	}
}

func TestRoomServiceJoinNonExistentRoom(t *testing.T) {
	db := setupTestDB(t)
	roomSvc := NewRoomService(db)
	ctx := context.Background()
	userID := int64(12002)
	createUserForTest(t, db, userID, "", "", 0)

	_, err := roomSvc.JoinRoom(ctx, userID, "non-existent-room")
	if err == nil {
		t.Fatal("Expected error for non-existent room")
	}
}

func TestRoomServiceGetRoomMembers(t *testing.T) {
	db := setupTestDB(t)
	roomSvc := NewRoomService(db)
	ctx := context.Background()

	user1 := int64(12003)
	user2 := int64(12004)
	createUserForTest(t, db, user1, "Laki-laki", "Teknik Sipil", 2022)
	createUserForTest(t, db, user2, "Perempuan", "Teknik Sipil", 2022)

	room, _ := roomSvc.CreateRoom(ctx, "Study Group", "Room belajar bareng sebelum UAS")
	_, _ = roomSvc.JoinRoom(ctx, user1, room.Slug)
	_, _ = roomSvc.JoinRoom(ctx, user2, room.Slug)

	members, roomName, err := roomSvc.GetRoomMembers(ctx, user1)
	if err != nil {
		t.Fatalf("GetRoomMembers failed: %v", err)
	}
	if len(members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(members))
	}
	if roomName != "Study Group" {
		t.Errorf("Expected room name 'Study Group', got '%s'", roomName)
	}
}

func TestRoomServiceGetMembersNotInRoom(t *testing.T) {
	db := setupTestDB(t)
	roomSvc := NewRoomService(db)
	ctx := context.Background()
	userID := int64(12005)
	createUserForTest(t, db, userID, "", "", 0)

	_, _, err := roomSvc.GetRoomMembers(ctx, userID)
	if err == nil {
		t.Fatal("Expected error for user not in any room")
	}
}

func TestRoomServiceGetActiveRooms(t *testing.T) {
	db := setupTestDB(t)
	roomSvc := NewRoomService(db)
	ctx := context.Background()

	_, _ = roomSvc.CreateRoom(ctx, "Room Alpha", "Room pertama untuk testing list rooms")
	_, _ = roomSvc.CreateRoom(ctx, "Room Beta", "Room kedua untuk testing list rooms")

	rooms, err := roomSvc.GetActiveRooms(ctx)
	if err != nil {
		t.Fatalf("GetActiveRooms failed: %v", err)
	}
	if len(rooms) != 2 {
		t.Errorf("Expected 2 rooms, got %d", len(rooms))
	}
}

func TestRoomServiceGetUserRoom(t *testing.T) {
	db := setupTestDB(t)
	roomSvc := NewRoomService(db)
	ctx := context.Background()
	userID := int64(12006)
	createUserForTest(t, db, userID, "Perempuan", "Akuntansi", 2023)

	room, _ := roomSvc.CreateRoom(ctx, "Chill Zone", "Room santai untuk curhat ringan")
	_, _ = roomSvc.JoinRoom(ctx, userID, room.Slug)

	userRoom, err := roomSvc.GetUserRoom(ctx, userID)
	if err != nil {
		t.Fatalf("GetUserRoom failed: %v", err)
	}
	if userRoom == nil || userRoom.Slug != room.Slug {
		t.Error("Expected user to be in the room they joined")
	}
}

func TestRoomServiceJoinSwitchesRoom(t *testing.T) {
	db := setupTestDB(t)
	roomSvc := NewRoomService(db)
	ctx := context.Background()
	userID := int64(12007)
	createUserForTest(t, db, userID, "Laki-laki", "Teknik Elektro", 2022)

	room1, _ := roomSvc.CreateRoom(ctx, "Room One", "Room pertama untuk test switch room")
	room2, _ := roomSvc.CreateRoom(ctx, "Room Two", "Room kedua untuk test switch room")

	_, _ = roomSvc.JoinRoom(ctx, userID, room1.Slug)
	_, _ = roomSvc.JoinRoom(ctx, userID, room2.Slug)

	userRoom, _ := roomSvc.GetUserRoom(ctx, userID)
	if userRoom == nil || userRoom.Slug != room2.Slug {
		t.Error("Expected user to be in room2 after switching")
	}
}

func TestRoomSlugGeneration(t *testing.T) {
	db := setupTestDB(t)
	roomSvc := NewRoomService(db)

	tests := []struct {
		name         string
		expectedSlug string
	}{
		{"Hello World", "hello-world"},
		{"Café & Lounge", "caf-lounge"},
		{"Test 123", "test-123"},
	}

	for _, tt := range tests {
		slug := roomSvc.createSlug(tt.name)
		if slug != tt.expectedSlug {
			t.Errorf("createSlug(%q) = %q, want %q", tt.name, slug, tt.expectedSlug)
		}
	}
}

func TestAuthServiceDuplicateRegistration(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.Config{OTPLength: 6, OTPExpiryMinutes: 10}
	authSvc := NewAuthService(db, &MockEmailSender{}, cfg)
	ctx := context.Background()

	userID := int64(13001)
	_, err := authSvc.RegisterUser(ctx, userID)
	if err != nil {
		t.Fatalf("First registration failed: %v", err)
	}

	user, err := authSvc.RegisterUser(ctx, userID)
	if err != nil {
		t.Fatalf("Duplicate registration returned error: %v", err)
	}
	if user == nil {
		t.Fatal("Expected non-nil user for duplicate registration")
	}
}

func TestAuthServiceIsVerified(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.Config{OTPLength: 6, OTPExpiryMinutes: 10}
	authSvc := NewAuthService(db, &MockEmailSender{}, cfg)
	ctx := context.Background()

	userID := int64(13002)
	_, _ = authSvc.RegisterUser(ctx, userID)

	verified, err := authSvc.IsVerified(ctx, userID)
	if err != nil {
		t.Fatalf("IsVerified failed: %v", err)
	}
	if verified {
		t.Error("New user should not be verified")
	}
}

func TestAuthServiceIsBanned(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.Config{OTPLength: 6, OTPExpiryMinutes: 10}
	authSvc := NewAuthService(db, &MockEmailSender{}, cfg)
	ctx := context.Background()

	userID := int64(13003)
	_, _ = authSvc.RegisterUser(ctx, userID)

	banned, err := authSvc.IsBanned(ctx, userID)
	if err != nil {
		t.Fatalf("IsBanned failed: %v", err)
	}
	if banned {
		t.Error("New user should not be banned")
	}
}

func TestAuthServiceStuEmail(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.Config{OTPLength: 6, OTPExpiryMinutes: 10}
	mockEmail := &MockEmailSender{}
	authSvc := NewAuthService(db, mockEmail, cfg)
	ctx := context.Background()

	userID := int64(13004)
	_, _ = authSvc.RegisterUser(ctx, userID)

	err := authSvc.InitiateVerification(ctx, userID, "test@stu.pnj.ac.id")
	if err != nil {
		t.Fatalf("InitiateVerification with stu.pnj.ac.id failed: %v", err)
	}
	if mockEmail.LastTo != "test@stu.pnj.ac.id" {
		t.Errorf("Expected email to test@stu.pnj.ac.id, got %s", mockEmail.LastTo)
	}
}

func TestAuthServiceEmailSenderError(t *testing.T) {
	db := setupTestDB(t)
	cfg := &config.Config{OTPLength: 6, OTPExpiryMinutes: 10}
	mockEmail := &MockEmailSender{Err: errMockSend}
	authSvc := NewAuthService(db, mockEmail, cfg)
	ctx := context.Background()

	userID := int64(13005)
	_, _ = authSvc.RegisterUser(ctx, userID)

	err := authSvc.InitiateVerification(ctx, userID, "test@mhsw.pnj.ac.id")
	if err == nil {
		t.Fatal("Expected error when email sender fails")
	}
}

var errMockSend = fmt.Errorf("email send failed")
