package session_test

import (
	"os"
	"tagg/session"
	"tagg/store"
	"testing"
	"time"
)

func setupTest(t *testing.T) (store.Store, *store.User, int64) {
	s, err := store.New("./test.db")
	if err != nil {
		t.Fatalf("error creating store: %v", err)
	}

	testUser := &store.User{
		GoogleID: "123456789",
		Email:    "test@example.com",
		Name:     "Test User",
		Picture:  "https://example.com/picture.jpg",
	}

	userID, err := s.CreateUser(testUser)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	testUser.ID = userID

	return s, testUser, userID
}

func cleanupTestDB(t *testing.T) {
	err := os.Remove("./test.db")
	if err != nil {
		t.Logf("Warning: Failed to remove test database: %v", err)
	}
}

func TestSessionCreation(t *testing.T) {
	s, _, userID := setupTest(t)
	defer cleanupTestDB(t)
	m := session.NewManager(s, 30, 15, false)

	session, err := m.CreateSession("test-token", userID)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if session.UserID != userID {
		t.Errorf("expected user ID %d, got %d", userID, session.UserID)
	}
}

func TestSessionValidation(t *testing.T) {
	s, user, _ := setupTest(t)
	defer cleanupTestDB(t)
	m := session.NewManager(s, 30, 15, false)

	session, err := m.CreateSession("test-token", user.ID)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	result, err := m.ValidateSessionToken("test-token")
	if err != nil {
		t.Fatalf("failed to validate valid session: %v", err)
	}
	if result.Session.ID != session.ID {
		t.Errorf("expected session ID %s, got %s", session.ID, result.Session.ID)
	}
	if result.User.ID != user.ID {
		t.Errorf("expected user ID %d, got %d", user.ID, result.User.ID)
	}

	result, err = m.ValidateSessionToken("invalid-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}

	result, err = m.ValidateSessionToken("")
	if err == nil {
		t.Error("expected error for empty token")
	}
}

func TestSessionExpiration(t *testing.T) {
	s, _, userID := setupTest(t)
	defer cleanupTestDB(t)
	m := session.NewManager(s, 1, 1, false)

	session, err := m.CreateSession("test-token", userID)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	session.ExpiresAt = time.Now().Add(-time.Hour * 25).Unix()
	err = s.RefreshSession(session.ID, session.ExpiresAt)
	if err != nil {
		t.Fatalf("failed to update session expiration: %v", err)
	}

	result, err := m.ValidateSessionToken("test-token")
	if err != nil {
		t.Fatalf("unexpected error validating expired session: %v", err)
	}
	if result != nil {
		t.Error("expected nil result for expired session")
	}
}

func TestSessionInvalidation(t *testing.T) {
	s, user, _ := setupTest(t)
	defer cleanupTestDB(t)
	m := session.NewManager(s, 30, 15, false)

	_, err := m.CreateSession("token1", user.ID)
	if err != nil {
		t.Fatalf("failed to create first session: %v", err)
	}
	_, err = m.CreateSession("token2", user.ID)
	if err != nil {
		t.Fatalf("failed to create second session: %v", err)
	}

	err = m.InvalidateUserSessions(user.ID)
	if err != nil {
		t.Fatalf("failed to invalidate user sessions: %v", err)
	}

	result, err := m.ValidateSessionToken("token1")
	if err == nil || result != nil {
		t.Error("session should be invalid after user sessions invalidation")
	}
	result, err = m.ValidateSessionToken("token2")
	if err == nil || result != nil {
		t.Error("session should be invalid after user sessions invalidation")
	}

	err = m.InvalidateUserSessions(999999)
	if err != nil {
		t.Error("invalidating non-existent sessions should not return error")
	}
}

func TestSessionRefresh(t *testing.T) {
	t.Run("successful refresh", func(t *testing.T) {
		s, _, userID := setupTest(t)
		defer cleanupTestDB(t)
		m := session.NewManager(s, 0, 0, false)
		session, err := m.CreateSession("test-token", userID)
		if err != nil {
			t.Fatalf("failed to create session: %v", err)
		}
		originalExpiresAt := session.ExpiresAt
		session.ExpiresAt = time.Now().Add(time.Hour * 25).Unix()
		err = s.RefreshSession(session.ID, session.ExpiresAt)
		if err != nil {
			t.Fatalf("failed to update session expiration: %v", err)
		}
		result, err := m.ValidateSessionToken("test-token")
		if err != nil {
			t.Fatalf("failed to validate session: %v", err)
		}
		if result.Session.ExpiresAt <= originalExpiresAt {
			t.Error("session expiration should be extended after refresh")
		}
	})

	t.Run("expired session", func(t *testing.T) {
		s, _, userID := setupTest(t)
		defer cleanupTestDB(t)
		m := session.NewManager(s, 0, 0, false)
		session, err := m.CreateSession("test-token", userID)
		if err != nil {
			t.Fatalf("failed to create session: %v", err)
		}

		session.ExpiresAt = time.Now().Add(-time.Hour).Unix()
		err = s.RefreshSession(session.ID, session.ExpiresAt)
		if err != nil {
			t.Fatalf("failed to update session expiration: %v", err)
		}

		result, err := m.ValidateSessionToken("test-token")
		if result != nil {
			t.Error("expired session should return nil result")
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		s, _, _ := setupTest(t)
		defer cleanupTestDB(t)
		m := session.NewManager(s, 0, 0, false)

		result, err := m.ValidateSessionToken("invalid-token")
		if err == nil {
			t.Error("expected error for invalid token")
		}
		if result != nil {
			t.Error("result should be nil for invalid token")
		}
	})

	t.Run("refresh threshold check", func(t *testing.T) {
		s, _, userID := setupTest(t)
		defer cleanupTestDB(t)
		m := session.NewManager(s, 30, 7, false)
		session, err := m.CreateSession("test-token", userID)
		if err != nil {
			t.Fatalf("failed to create session: %v", err)
		}

		thresholdTime := time.Now().Add(time.Hour * 24 * 6)
		session.ExpiresAt = thresholdTime.Unix()
		err = s.RefreshSession(session.ID, session.ExpiresAt)
		if err != nil {
			t.Fatalf("failed to update session expiration: %v", err)
		}

		result, err := m.ValidateSessionToken("test-token")
		if err != nil {
			t.Fatalf("failed to validate session: %v", err)
		}
		if result.Session.ExpiresAt <= thresholdTime.Unix() {
			t.Error("session should be refreshed when within threshold")
		}
	})
}
