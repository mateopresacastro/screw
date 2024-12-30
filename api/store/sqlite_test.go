package store

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func setupTestDB(t *testing.T) Store {
	tmpfile := "./test.db"
	os.Remove(tmpfile)

	store, err := newSQLiteStore(tmpfile)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	return store
}

func cleanupTestDB(t *testing.T) {
	err := os.Remove("./test.db")
	if err != nil {
		t.Logf("Warning: Failed to remove test database: %v", err)
	}
}

func TestCreateUser(t *testing.T) {
	store := setupTestDB(t)
	defer cleanupTestDB(t)

	testUser := &User{
		GoogleID: "123456789",
		Email:    "test@example.com",
		Name:     "Test User",
		Picture:  "https://example.com/picture.jpg",
	}

	id, err := store.CreateUser(testUser)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	if id <= 0 {
		t.Errorf("Expected positive user ID, got %d", id)
	}

	_, err = store.CreateUser(testUser)
	if err == nil {
		t.Error("Expected error when creating duplicate user, got nil")
	}
}

func TestGetUserById(t *testing.T) {
	store := setupTestDB(t)
	defer cleanupTestDB(t)

	testUser := &User{
		GoogleID: "123456789",
		Email:    "test@example.com",
		Name:     "Test User",
		Picture:  "https://example.com/picture.jpg",
	}

	id, err := store.CreateUser(testUser)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	user, err := store.GetUserByGoogleId(testUser.GoogleID)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if user.ID != id {
		t.Errorf("Expected user ID %d, got %d", id, user.ID)
	}
	if user.GoogleID != testUser.GoogleID {
		t.Errorf("Expected GoogleID %s, got %s", testUser.GoogleID, user.GoogleID)
	}
	if user.Email != testUser.Email {
		t.Errorf("Expected Email %s, got %s", testUser.Email, user.Email)
	}
	if user.Name != testUser.Name {
		t.Errorf("Expected Name %s, got %s", testUser.Name, user.Name)
	}
	if user.Picture != testUser.Picture {
		t.Errorf("Expected Picture %s, got %s", testUser.Picture, user.Picture)
	}

	_, err = store.GetUserByGoogleId("not real")
	if err == nil {
		t.Error("Expected error when getting non-existent user, got nil")
	}
}

func TestDeleteUser(t *testing.T) {
	store := setupTestDB(t)
	defer cleanupTestDB(t)

	testUser := &User{
		GoogleID: "123456789",
		Email:    "test@example.com",
		Name:     "Test User",
		Picture:  "https://example.com/picture.jpg",
	}

	id, err := store.CreateUser(testUser)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	err = store.DeleteUser(id)
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	_, err = store.GetUserByGoogleId(testUser.GoogleID)
	if err == nil {
		t.Error("Expected error when getting deleted user, got nil")
	}

	err = store.DeleteUser(4444)
	if err == nil {
		t.Error("Expected error when deleting non-existent user, got nil")
	}
}

func TestConcurrentAccess(t *testing.T) {
	store := setupTestDB(t)
	defer cleanupTestDB(t)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(i int) {
			testUser := &User{
				GoogleID: fmt.Sprintf("user%d", i),
				Email:    fmt.Sprintf("user%d@example.com", i),
				Name:     fmt.Sprintf("Test User %d", i),
				Picture:  "https://example.com/picture.jpg",
			}
			_, err := store.CreateUser(testUser)
			if err != nil {
				t.Errorf("Failed to create user in goroutine: %v", err)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestCreateSession(t *testing.T) {
	store := setupTestDB(t)
	defer cleanupTestDB(t)

	testUser := &User{
		GoogleID: "123456789",
		Email:    "test@example.com",
		Name:     "Test User",
		Picture:  "https://example.com/picture.jpg",
	}
	userId, err := store.CreateUser(testUser)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	sessionId := "12345"
	expiresAt := time.Now().Add(24 * time.Hour).Unix()
	session, err := store.CreateSession(sessionId, userId, expiresAt)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	if session.ID != sessionId {
		t.Errorf("Expected session ID %s, got %s", sessionId, session.ID)
	}
	if session.UserID != userId {
		t.Errorf("Expected user ID %d, got %d", userId, session.UserID)
	}
	if session.ExpiresAt != expiresAt {
		t.Errorf("Expected expires at %d, got %d", expiresAt, session.ExpiresAt)
	}

	_, err = store.CreateSession("123453", int64(9999), expiresAt)
	if err == nil {
		t.Error("Expected error when creating session for non-existent user, got nil")
	}

	_, err = store.CreateSession(sessionId, userId, expiresAt)
	if err == nil {
		t.Error("Expected error when creating duplicate session, got nil")
	}
}

func TestDeleteSessionByUserId(t *testing.T) {
	store := setupTestDB(t)
	defer cleanupTestDB(t)

	testUser := &User{
		GoogleID: "123456789",
		Email:    "test@example.com",
		Name:     "Test User",
		Picture:  "https://example.com/picture.jpg",
	}
	userId, err := store.CreateUser(testUser)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	sessionId := "1234"
	expiresAt := time.Now().Add(24 * time.Hour).Unix()
	_, err = store.CreateSession(sessionId, userId, expiresAt)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	err = store.DeleteSessionByUserId(userId)
	if err != nil {
		t.Fatalf("Failed to delete session by user ID: %v", err)
	}

	_, _, err = store.GetSessionAndUserBySessionId(sessionId)
	if err == nil {
		t.Error("Expected error when getting deleted session, got nil")
	}

	err = store.DeleteSessionByUserId(int64(9999))
	if err != nil {
		t.Error("Expected no error when deleting sessions for non-existent user")
	}
}

func TestDeleteSessionBySessionId(t *testing.T) {
	store := setupTestDB(t)
	defer cleanupTestDB(t)

	testUser := &User{
		GoogleID: "123456789",
		Email:    "test@example.com",
		Name:     "Test User",
		Picture:  "https://example.com/picture.jpg",
	}
	userId, err := store.CreateUser(testUser)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	sessionId := "12345"
	expiresAt := time.Now().Add(24 * time.Hour).Unix()
	_, err = store.CreateSession(sessionId, userId, expiresAt)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	err = store.DeleteSessionBySessionId(sessionId)
	if err != nil {
		t.Fatalf("Failed to delete session by session ID: %v", err)
	}

	_, _, err = store.GetSessionAndUserBySessionId(sessionId)
	if err == nil {
		t.Error("Expected error when getting deleted session, got nil")
	}

	err = store.DeleteSessionBySessionId("000000000")
	if err != nil {
		t.Error("Expected no error when deleting non-existent session")
	}
}

func TestGetSessionAndUserBySessionId(t *testing.T) {
	store := setupTestDB(t)
	defer cleanupTestDB(t)

	testUser := &User{
		GoogleID: "123456789",
		Email:    "test@example.com",
		Name:     "Test User",
		Picture:  "https://example.com/picture.jpg",
	}
	userId, err := store.CreateUser(testUser)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	sessionId := "12345"
	expiresAt := time.Now().Add(24 * time.Hour).Unix()
	_, err = store.CreateSession(sessionId, userId, expiresAt)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	session, user, err := store.GetSessionAndUserBySessionId(sessionId)
	if err != nil {
		t.Fatalf("Failed to get session and user: %v", err)
	}

	if session.ID != sessionId {
		t.Errorf("Expected session ID %s, got %s", sessionId, session.ID)
	}
	if session.UserID != userId {
		t.Errorf("Expected user ID %d, got %d", userId, session.UserID)
	}
	if session.ExpiresAt != expiresAt {
		t.Errorf("Expected expires at %d, got %d", expiresAt, session.ExpiresAt)
	}

	if user.ID != userId {
		t.Errorf("Expected user ID %d, got %d", userId, user.ID)
	}
	if user.GoogleID != testUser.GoogleID {
		t.Errorf("Expected GoogleID %s, got %s", testUser.GoogleID, user.GoogleID)
	}
	if user.Email != testUser.Email {
		t.Errorf("Expected Email %s, got %s", testUser.Email, user.Email)
	}

	_, _, err = store.GetSessionAndUserBySessionId("000000")
	if err == nil {
		t.Error("Expected error when getting non-existent session, got nil")
	}
}

func TestRefreshSession(t *testing.T) {
	store := setupTestDB(t)
	defer cleanupTestDB(t)

	testUser := &User{
		GoogleID: "123456789",
		Email:    "test@example.com",
		Name:     "Test User",
		Picture:  "https://example.com/picture.jpg",
	}
	userId, err := store.CreateUser(testUser)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	sessionId := "12345"
	expiresAt := time.Now().Add(24 * time.Hour).Unix()
	_, err = store.CreateSession(sessionId, userId, expiresAt)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	newExpiresAt := time.Now().Add(48 * time.Hour).Unix()
	err = store.RefreshSession(sessionId, newExpiresAt)
	if err != nil {
		t.Fatalf("Failed to refresh session: %v", err)
	}

	session, _, err := store.GetSessionAndUserBySessionId(sessionId)
	if err != nil {
		t.Fatalf("Failed to get refreshed session: %v", err)
	}
	if session.ExpiresAt != newExpiresAt {
		t.Errorf("Expected expires at %d, got %d", newExpiresAt, session.ExpiresAt)
	}

	err = store.RefreshSession("000000000", newExpiresAt)
	if err != nil {
		t.Error("Expected no error when refreshing non-existent session")
	}
}

func TestConcurrentSessionOperations(t *testing.T) {
	store := setupTestDB(t)
	defer cleanupTestDB(t)

	testUser := &User{
		GoogleID: "123456789",
		Email:    "test@example.com",
		Name:     "Test User",
		Picture:  "https://example.com/picture.jpg",
	}
	userId, err := store.CreateUser(testUser)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	done := make(chan bool)
	sessionCount := 10

	baseTime := time.Now()
	for i := 0; i < sessionCount; i++ {
		go func(i int) {
			sessionId := fmt.Sprintf("sessionId-%d", i)
			expiresAt := baseTime.Add(time.Duration(i) * time.Hour).Unix()
			_, err := store.CreateSession(sessionId, userId, expiresAt)
			if err != nil {
				t.Errorf("Failed to create session in goroutine: %v", err)
			}

			newExpiresAt := baseTime.Add(time.Duration(i+24) * time.Hour).Unix()
			err = store.RefreshSession(sessionId, newExpiresAt)
			if err != nil {
				t.Errorf("Failed to refresh session in goroutine: %v", err)
			}

			done <- true
		}(i)
	}

	for i := 0; i < sessionCount; i++ {
		<-done
	}
}
