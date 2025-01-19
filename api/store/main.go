package store

type Store interface {
	CreateUser(user *User) (int64, error)
	UserByGoogleID(googleID string) (*User, error)
	DeleteUser(userID int64) error
	CreateSession(sessionID string, userID int64, expiresAt int64) (*Session, error)
	DeleteSessionByUserID(userID int64) (err error)
	DeleteSessionBySessionID(sessionID string) (err error)
	SessionAndUserBySessionID(sessionID string) (*Session, *User, error)
	RefreshSession(sessionID string, newExpiresAt int64) error
}

func New(dbPath string) (Store, error) {
	return newSQLiteStore(dbPath)
}
