package session

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"tagg/store"
	"time"
)

const (
	sessionCookieName = "session"
	oneDayInHours     = 24
)

type Manager struct {
	store                   store.Store
	sessionExpirationInDays int64
	refreshThresholdDays    int64
	isProd                  bool
}

func NewManager(store store.Store, sessionExpirationInDays int64, refreshThresholdDays int64, isProd bool) *Manager {
	return &Manager{
		store:                   store,
		sessionExpirationInDays: sessionExpirationInDays,
		refreshThresholdDays:    refreshThresholdDays,
		isProd:                  isProd,
	}
}

func (m *Manager) CreateSession(token string, userId int64) (*store.Session, error) {
	sessionId := generateSessionId(token)
	expiresAt := m.newExpiresAt()
	session, err := m.store.CreateSession(sessionId, userId, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("error creating session: %w", err)
	}
	return session, nil
}

func generateSessionId(token string) string {
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

	sessionId := generateSessionId(token)
	session, user, err := m.store.GetSessionAndUserBySessionId(sessionId)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	expiresAt := time.Unix(session.ExpiresAt, 0)

	if now.After(expiresAt) {
		if err := m.store.DeleteSessionBySessionId(session.ID); err != nil {
			return nil, fmt.Errorf("error deleting expired session: %w", err)
		}
		return nil, nil
	}

	thresholdDuration := time.Duration(m.refreshThresholdDays) * oneDayInHours * time.Hour
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

func (m *Manager) InvalidateSession(sessionId string) error {
	return m.store.DeleteSessionBySessionId(sessionId)
}

func (m *Manager) InvalidateUserSessions(userId int64) error {
	return m.store.DeleteSessionByUserId(userId)
}

func SetSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time, isProd bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		HttpOnly: true,
		Path:     "/",
		Secure:   isProd,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
	})
}

func DeleteSessionCookie(w http.ResponseWriter, isProd bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		HttpOnly: true,
		Path:     "/",
		Secure:   isProd,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}
