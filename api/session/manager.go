package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"tagg/store"
	"time"
)

const ()

const (
	SessionContextKey = "session"
	sessionCookieName = "session"
	oneDayInHours     = 24
)

type Manager struct {
	store                   store.Store
	sessionExpirationInDays int64
	refreshThresholdInDays  int64
	isProd                  bool
}

func NewManager(store store.Store, sessionExpirationInDays int64, refreshThresholdInDays int64, isProd bool) *Manager {
	return &Manager{
		store:                   store,
		sessionExpirationInDays: sessionExpirationInDays,
		refreshThresholdInDays:  refreshThresholdInDays,
		isProd:                  isProd,
	}
}

func (m *Manager) CreateSession(token string, userID int64) (*store.Session, error) {
	err := m.InvalidateUserSessions(userID)
	if err != nil {
		slog.Warn("error deleting old sessions", "err", err)
	}

	sessionID := generateSessionID(token)
	expiresAt := m.newExpiresAt()
	session, err := m.store.CreateSession(sessionID, userID, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("error creating session: %w", err)
	}
	return session, nil
}

func generateSessionID(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

type SessionValidationResult struct {
	Session *store.Session `json:"session"`
	User    *store.User    `json:"user"`
}

func (m *Manager) newExpiresAt() int64 {
	return time.Now().Add(time.Duration(m.sessionExpirationInDays) * oneDayInHours * time.Hour).Unix()
}

func (m *Manager) ValidateSessionToken(token string) (*SessionValidationResult, error) {
	if token == "" {
		return nil, fmt.Errorf("empty session token")
	}

	sessionID := generateSessionID(token)
	session, user, err := m.store.SessionAndUserBySessionID(sessionID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	expiresAt := time.Unix(session.ExpiresAt, 0)

	if now.After(expiresAt) {
		if err := m.store.DeleteSessionBySessionID(session.ID); err != nil {
			return nil, fmt.Errorf("error deleting expired session: %w", err)
		}
		return nil, nil
	}

	thresholdDuration := time.Duration(m.refreshThresholdInDays) * oneDayInHours * time.Hour
	thresholdTime := expiresAt.Add(-thresholdDuration)

	if now.After(thresholdTime) {
		newExpiresAt := m.newExpiresAt()
		err = m.store.RefreshSession(session.ID, newExpiresAt)
		if err != nil {
			return nil, fmt.Errorf("error refreshing session: %w", err)
		}
		session.ExpiresAt = newExpiresAt
	}

	return &SessionValidationResult{Session: session, User: user}, nil
}

func (m *Manager) InvalidateSession(sessionID string) error {
	return m.store.DeleteSessionBySessionID(sessionID)
}

func (m *Manager) InvalidateUserSessions(userID int64) error {
	return m.store.DeleteSessionByUserID(userID)
}

func (m *Manager) SetSessionCookie(w http.ResponseWriter, token string, expiresAt int64) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		HttpOnly: true,
		Path:     "/",
		Secure:   m.isProd,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(expiresAt, 0),
	})
}

func (m *Manager) DeleteSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		HttpOnly: true,
		Path:     "/",
		Secure:   m.isProd,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func (m *Manager) GetCurrentSession(r *http.Request) (*SessionValidationResult, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, fmt.Errorf("error getting session cookie: %w", err)
	}

	if cookie.Value == "" {
		return nil, fmt.Errorf("session cookie is empty")
	}

	result, err := m.ValidateSessionToken(cookie.Value)
	if err != nil {
		return nil, fmt.Errorf("error validating session token: %w", err)
	}

	return result, nil
}

func (m *Manager) GenerateRandomSessionToken() (string, error) {
	bytes := make([]byte, 25)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("error generating random bytes: %w", err)
	}
	token := strings.ToLower(base32.StdEncoding.EncodeToString(bytes))
	return token, nil
}

func GetSessionFromContext(ctx context.Context) (*SessionValidationResult, bool) {
	session, ok := ctx.Value(SessionContextKey).(*SessionValidationResult)
	return session, ok
}

func (m *Manager) HandleCurrentSession(w http.ResponseWriter, r *http.Request) {
	result, ok := GetSessionFromContext(r.Context())
	if !ok {
		slog.Error("no session data on context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	response := struct {
		Name    string `json:"name"`
		Email   string `json:"email"`
		Picture string `json:"picture"`
	}{
		Name:    result.User.Name,
		Email:   result.User.Email,
		Picture: result.User.Picture,
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("error encoding response", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
}

func (m *Manager) HandleLogout(w http.ResponseWriter, r *http.Request) {
	result, ok := GetSessionFromContext(r.Context())
	if !ok {
		slog.Error("no session data on context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err := m.InvalidateSession(result.Session.ID)
	if err != nil {
		slog.Error("error getting session", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	m.DeleteSessionCookie(w)
	return
}
