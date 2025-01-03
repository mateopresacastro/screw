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

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("error enabling WAL mode: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("error enabling foreign keys: %w", err)
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

	_, err = s.db.Exec(`
		CREATE TABlE IF NOT EXISTS tag (
			id TEXT NOT NULL PRIMARY KEY,
		    user_id INTEGER NOT NULL REFERENCES user(id),
			file_path TEXT NOT NULL UNIQUE
		)
	`)
	if err != nil {
		return fmt.Errorf("error creating tag table: %w", err)
	}

	return nil
}

func (s *sqliteStore) CreateUser(user *User) (int64, error) {
	slog.Info("DB: creating user", "user", user)
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
	slog.Info("DB: user created")
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

	slog.Info("DB: got user by google id ", "user", user)
	return user, nil
}

func (s *sqliteStore) DeleteUser(userID int64) error {
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

	slog.Info("DB: deleted user")
	return nil
}

func (s *sqliteStore) CreateSession(sessionID string, userID int64, expiresAt int64) (*Session, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()

	var exists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM user WHERE id = ?)", userID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("error checking user existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	query := "INSERT INTO session (id, user_id, expires_at) VALUES (?, ?, ?)"
	_, err = tx.Exec(query, sessionID, userID, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("error creating session: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("error committing transaction: %w", err)
	}

	session := &Session{
		ID:        sessionID,
		UserID:    userID,
		ExpiresAt: expiresAt,
	}
	slog.Info("DB: session created")
	return session, nil
}

func (s *sqliteStore) DeleteSessionByUserID(userID int64) error {
	_, err := s.db.Exec("DELETE FROM session WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("error deleting session by userID: %w", err)
	}

	slog.Info("DB: all old session deleted for user")
	return nil
}

func (s *sqliteStore) DeleteSessionBySessionID(sessionID string) error {
	_, err := s.db.Exec("DELETE FROM session WHERE id = ?", sessionID)
	if err != nil {
		return fmt.Errorf("error deleting session by sessionID: %w", err)
	}
	slog.Info("DB: session deleted")
	return nil
}

func (s *sqliteStore) SessionAndUserBySessionID(sessionID string) (*Session, *User, error) {
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

	slog.Info("DB: got session")
	return session, user, nil
}

func (s *sqliteStore) RefreshSession(sessionID string, newExpiresAt int64) error {
	query := "UPDATE session SET expires_at = ? WHERE id = ?"
	_, err := s.db.Exec(query, newExpiresAt, sessionID)
	if err != nil {
		return fmt.Errorf("error updating session: %w", err)
	}
	slog.Info("DB: session rereshed")
	return nil
}

func (s *sqliteStore) DeleteTag(tagID string) error {
	_, err := s.db.Exec("DELETE FROM tag WHERE id = ?", tagID)
	if err != nil {
		return fmt.Errorf("error deleting tag: %w", err)
	}
	slog.Info("DB: Tag deleted")
	return nil
}

func (s *sqliteStore) CreateTag(tag *Tag) error {
	slog.Info("DB: creating tag", "tag", tag)
	query := `
        INSERT INTO tag (id, user_id, file_path)
        VALUES (?, ?, ?)
    `
	var result sql.Result
	result, err := s.db.Exec(query, tag.ID, tag.UserID, tag.FilePath)
	if err != nil {
		return fmt.Errorf("error creating tag: %w", err)
	}

	_, err = result.LastInsertId()
	if err != nil {
		return fmt.Errorf("error getting last insert id: %w", err)
	}
	slog.Info("DB: tag created")
	return nil
}

func (s *sqliteStore) TagByUserID(userID int64) (*Tag, error) {
	tag := &Tag{}
	err := s.db.QueryRow(`
			SELECT id, file_path, user_id
			FROM tag
			WHERE user_id = ?
		`, userID).Scan(&tag.ID, &tag.FilePath, &tag.UserID)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tag not found: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("error getting tag: %w", err)
	}

	slog.Info("DB: got tag by userID", "tag", tag)
	return tag, nil
}
