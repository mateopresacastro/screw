package session_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"screw/session"
	"screw/store"
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
	if err := os.Remove("./test.db"); err != nil && !os.IsNotExist(err) {
		t.Fatalf("Failed to remove test database: %v", err)
	}
}

func TestSessionCreation(t *testing.T) {
	s, _, userID := setupTest(t)
	defer cleanupTestDB(t)
	m := session.NewManager(s, 30, 15)

	w := httptest.NewRecorder()
	token, err := m.CreateSession(w, userID)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != session.SessionCookieName {
		t.Errorf("expected cookie name %q, got %q", session.SessionCookieName, cookie.Name)
	}
	if cookie.Value != token {
		t.Errorf("expected cookie value %q, got %q", token, cookie.Value)
	}
}

func TestSessionValidation(t *testing.T) {
	s, user, _ := setupTest(t)
	defer cleanupTestDB(t)
	m := session.NewManager(s, 30, 15)

	t.Run("valid token", func(t *testing.T) {
		w := httptest.NewRecorder()
		token, err := m.CreateSession(w, user.ID)
		if err != nil {
			t.Fatalf("failed to create session: %v", err)
		}

		result, err := m.ValidateSessionToken(token)
		if err != nil {
			t.Fatalf("failed to validate valid session: %v", err)
		}
		if result.User.ID != user.ID {
			t.Errorf("expected user ID %d, got %d", user.ID, result.User.ID)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		result, err := m.ValidateSessionToken("invalid-token")
		if err == nil {
			t.Error("expected error for invalid token")
		}
		if result != nil {
			t.Error("expected nil result for invalid token")
		}
	})

	t.Run("empty token", func(t *testing.T) {
		result, err := m.ValidateSessionToken("")
		if err == nil {
			t.Error("expected error for empty token")
		}
		if result != nil {
			t.Error("expected nil result for empty token")
		}
	})
}

func TestSessionExpiration(t *testing.T) {
	s, _, userID := setupTest(t)
	defer cleanupTestDB(t)
	m := session.NewManager(s, 1, 1)

	w := httptest.NewRecorder()
	token, err := m.CreateSession(w, userID)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	result, err := m.ValidateSessionToken(token)
	if err != nil {
		t.Fatalf("failed to validate session: %v", err)
	}

	result.Session.ExpiresAt = time.Now().Add(-time.Hour).Unix()
	err = s.RefreshSession(result.Session.ID, result.Session.ExpiresAt)
	if err != nil {
		t.Fatalf("failed to update session expiration: %v", err)
	}

	result, err = m.ValidateSessionToken(token)
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
	m := session.NewManager(s, 30, 15)

	t.Run("invalidate user sessions", func(t *testing.T) {
		w1 := httptest.NewRecorder()
		token1, err := m.CreateSession(w1, user.ID)
		if err != nil {
			t.Fatalf("failed to create first session: %v", err)
		}

		w2 := httptest.NewRecorder()
		token2, err := m.CreateSession(w2, user.ID)
		if err != nil {
			t.Fatalf("failed to create second session: %v", err)
		}

		err = m.InvalidateUserSessions(user.ID)
		if err != nil {
			t.Fatalf("failed to invalidate user sessions: %v", err)
		}

		result, err := m.ValidateSessionToken(token1)
		if err == nil || result != nil {
			t.Error("session should be invalid after user sessions invalidation")
		}

		result, err = m.ValidateSessionToken(token2)
		if err == nil || result != nil {
			t.Error("session should be invalid after user sessions invalidation")
		}
	})

	t.Run("invalidate non-existent sessions", func(t *testing.T) {
		err := m.InvalidateUserSessions(999999)
		if err != nil {
			t.Error("invalidating non-existent sessions should not return error")
		}
	})
}

func TestSessionRefresh(t *testing.T) {
	t.Run("within refresh threshold", func(t *testing.T) {
		s, _, userID := setupTest(t)
		defer cleanupTestDB(t)
		m := session.NewManager(s, 30, 7)

		w := httptest.NewRecorder()
		token, err := m.CreateSession(w, userID)
		if err != nil {
			t.Fatalf("failed to create session: %v", err)
		}

		result, err := m.ValidateSessionToken(token)
		if err != nil {
			t.Fatalf("failed to validate session: %v", err)
		}

		// Set expiration within refresh threshold (7 days)
		thresholdTime := time.Now().Add(time.Hour * 24 * 6)
		result.Session.ExpiresAt = thresholdTime.Unix()
		err = s.RefreshSession(result.Session.ID, result.Session.ExpiresAt)
		if err != nil {
			t.Fatalf("failed to update session expiration: %v", err)
		}

		refreshedResult, err := m.ValidateSessionToken(token)
		if err != nil {
			t.Fatalf("failed to validate session: %v", err)
		}
		if refreshedResult.Session.ExpiresAt <= thresholdTime.Unix() {
			t.Error("session should be refreshed when within threshold")
		}
	})

	t.Run("outside refresh threshold", func(t *testing.T) {
		s, _, userID := setupTest(t)
		defer cleanupTestDB(t)
		m := session.NewManager(s, 30, 7)

		w := httptest.NewRecorder()
		token, err := m.CreateSession(w, userID)
		if err != nil {
			t.Fatalf("failed to create session: %v", err)
		}

		result, err := m.ValidateSessionToken(token)
		if err != nil {
			t.Fatalf("failed to validate session: %v", err)
		}

		// Set expiration outside refresh threshold
		originalTime := time.Now().Add(time.Hour * 24 * 20)
		result.Session.ExpiresAt = originalTime.Unix()
		err = s.RefreshSession(result.Session.ID, result.Session.ExpiresAt)
		if err != nil {
			t.Fatalf("failed to update session expiration: %v", err)
		}

		validatedResult, err := m.ValidateSessionToken(token)
		if err != nil {
			t.Fatalf("failed to validate session: %v", err)
		}
		if validatedResult.Session.ExpiresAt != originalTime.Unix() {
			t.Error("session should not be refreshed when outside threshold")
		}
	})
}

func TestGetCurrentSession(t *testing.T) {
	s, _, userID := setupTest(t)
	defer cleanupTestDB(t)
	m := session.NewManager(s, 30, 15)

	t.Run("valid session", func(t *testing.T) {
		w := httptest.NewRecorder()
		token, err := m.CreateSession(w, userID)
		if err != nil {
			t.Fatalf("failed to create session: %v", err)
		}

		r := httptest.NewRequest("GET", "/", nil)
		r.AddCookie(&http.Cookie{
			Name:  session.SessionCookieName,
			Value: token,
		})

		result, err := m.GetCurrentSession(r)
		if err != nil {
			t.Fatalf("failed to get current session: %v", err)
		}
		if result.User.ID != userID {
			t.Errorf("expected user ID %d, got %d", userID, result.User.ID)
		}
	})

	t.Run("no cookie", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		result, err := m.GetCurrentSession(r)
		if err == nil {
			t.Error("expected error for missing cookie")
		}
		if result != nil {
			t.Error("expected nil result for missing cookie")
		}
	})

	t.Run("empty cookie value", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		r.AddCookie(&http.Cookie{
			Name:  session.SessionCookieName,
			Value: "",
		})

		result, err := m.GetCurrentSession(r)
		if err == nil {
			t.Error("expected error for empty cookie value")
		}
		if result != nil {
			t.Error("expected nil result for empty cookie value")
		}
	})
}

func TestHandleCurrentSession(t *testing.T) {
	s, user, _ := setupTest(t)
	defer cleanupTestDB(t)
	m := session.NewManager(s, 30, 15)

	t.Run("valid session", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		ctx := context.WithValue(r.Context(), session.SessionContextKey, &session.SessionValidationResult{
			User:    user,
			Session: &store.Session{ID: "test", UserID: user.ID},
		})
		r = r.WithContext(ctx)

		err := m.HandleCurrentSession(w, r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var response struct {
			Name    string `json:"name"`
			Email   string `json:"email"`
			Picture string `json:"picture"`
		}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response.Name != user.Name {
			t.Errorf("expected name %q, got %q", user.Name, response.Name)
		}
		if response.Email != user.Email {
			t.Errorf("expected email %q, got %q", user.Email, response.Email)
		}
		if response.Picture != user.Picture {
			t.Errorf("expected picture %q, got %q", user.Picture, response.Picture)
		}
	})

	t.Run("no session in context", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		err := m.HandleCurrentSession(w, r)
		if err == nil {
			t.Error("expected error for missing session in context")
		}
	})
}

func TestHandleLogout(t *testing.T) {
	s, user, _ := setupTest(t)
	defer cleanupTestDB(t)
	m := session.NewManager(s, 30, 15)

	t.Run("successful logout", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		ctx := context.WithValue(r.Context(), session.SessionContextKey, &session.SessionValidationResult{
			User:    user,
			Session: &store.Session{ID: "test", UserID: user.ID},
		})
		r = r.WithContext(ctx)

		err := m.HandleLogout(w, r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cookies := w.Result().Cookies()
		if len(cookies) != 1 {
			t.Fatalf("expected 1 cookie, got %d", len(cookies))
		}

		cookie := cookies[0]
		if cookie.Name != session.SessionCookieName {
			t.Errorf("expected cookie name %q, got %q", session.SessionCookieName, cookie.Name)
		}
		if cookie.Value != "" {
			t.Error("cookie value should be empty")
		}
		if cookie.MaxAge != -1 {
			t.Error("cookie should be expired")
		}
	})

	t.Run("no session in context", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		err := m.HandleLogout(w, r)
		if err == nil {
			t.Error("expected error for missing session in context")
		}
	})
}
