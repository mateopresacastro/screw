package store

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type sqliteStore struct {
	db    *sql.DB
	mutex sync.Mutex
}

func newSQLiteStore(path string) (Store, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	store := &sqliteStore{
		db: db,
	}

	if err := store.initializeTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("error initializing tables: %w", err)
	}

	return store, nil
}

func (s *sqliteStore) initializeTables() error {
	_, err := s.db.Exec(`
        CREATE TABLE IF NOT EXISTS user (
            id INTEGER NOT NULL PRIMARY KEY,
            google_id TEXT NOT NULL UNIQUE,
            email TEXT NOT NULL UNIQUE,
            name TEXT NOT NULL,
            picture TEXT NOT NULL
        )
    `)
	if err != nil {
		return fmt.Errorf("error creating user table: %w", err)
	}

	_, err = s.db.Exec(`
        CREATE INDEX IF NOT EXISTS google_id_index ON user(google_id)
    `)
	if err != nil {
		return fmt.Errorf("error creating google_id index: %w", err)
	}

	_, err = s.db.Exec(`
        CREATE TABLE IF NOT EXISTS session (
            id TEXT NOT NULL PRIMARY KEY,
            user_id INTEGER NOT NULL REFERENCES user(id),
            expires_at INTEGER NOT NULL
        )
    `)
	if err != nil {
		return fmt.Errorf("error creating session table: %w", err)
	}

	return nil
}

func (s *sqliteStore) CreateUser(user *User) (int64, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	slog.Info("db: creating user", "user", user)
	query := `
        INSERT INTO user (google_id, email, name, picture)
        VALUES (?, ?, ?, ?)
    `
	var result sql.Result
	result, err := s.db.Exec(query, user.GoogleID, user.Email, user.Name, user.Picture)
	if err != nil {
		return 0, fmt.Errorf("error creating user: %w", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("error getting last insert id: %w", err)
	}
	slog.Info("db: user created")
	return userID, nil
}

var ErrUserNotFound = errors.New("user not found")

func (s *sqliteStore) UserByGoogleID(googleID string) (*User, error) {
	user := &User{}
	err := s.db.QueryRow(`
        SELECT id, google_id, email, name, picture
        FROM user
        WHERE google_id = ?
    `, googleID).Scan(&user.ID, &user.GoogleID, &user.Email, &user.Name, &user.Picture)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error getting user: %w", err)
	}

	slog.Info("db: got user by google id ", "user", user)
	return user, nil
}

func (s *sqliteStore) DeleteUser(userID int64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	result, err := s.db.Exec("DELETE FROM user WHERE id = ?", userID)
	if err != nil {
		return fmt.Errorf("error deleting user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	slog.Info("db: deleted user")
	return nil
}

func (s *sqliteStore) CreateSession(sessionID string, userID int64, expiresAt int64) (*Session, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM user WHERE id = ?)", userID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("error checking user existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	query := "INSERT INTO session (id, user_id, expires_at) VALUES (?, ?, ?)"
	_, err = s.db.Exec(query, sessionID, userID, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("error creating session: %w", err)
	}

	session := &Session{
		ID:        sessionID,
		UserID:    userID,
		ExpiresAt: expiresAt,
	}
	slog.Info("db: session created")
	return session, nil
}

func (s *sqliteStore) DeleteSessionByUserID(userID int64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	_, err := s.db.Exec("DELETE FROM session WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("error deleting session by userID: %w", err)
	}

	slog.Info("db: all old session deleted for user")
	return nil
}

func (s *sqliteStore) DeleteSessionBySessionID(sessionID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	_, err := s.db.Exec("DELETE FROM session WHERE id = ?", sessionID)
	if err != nil {
		return fmt.Errorf("error deleting session by sessionID: %w", err)
	}
	slog.Info("db: session deleted")
	return nil
}

func (s *sqliteStore) SessionAndUserBySessionID(sessionID string) (*Session, *User, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	session := &Session{}
	user := &User{}

	query := `
        SELECT session.id, session.user_id, session.expires_at, user.id, user.google_id, user.email, user.name, user.picture
        FROM session
        INNER JOIN user ON session.user_id = user.id
        WHERE session.id = ?
    `
	err := s.db.QueryRow(query, sessionID).Scan(
		&session.ID,
		&session.UserID,
		&session.ExpiresAt,
		&user.ID,
		&user.GoogleID,
		&user.Email,
		&user.Name,
		&user.Picture,
	)

	if err == sql.ErrNoRows {
		return nil, nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, nil, fmt.Errorf("error getting session and user: %w", err)
	}

	slog.Info("db: got session")
	return session, user, nil
}

func (s *sqliteStore) RefreshSession(sessionID string, newExpiresAt int64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	query := "UPDATE session SET expires_at = ? WHERE id = ?"
	_, err := s.db.Exec(query, newExpiresAt, sessionID)
	if err != nil {
		return fmt.Errorf("error updating session: %w", err)
	}
	slog.Info("db: session rereshed")
	return nil
}
