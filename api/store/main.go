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
	CreateTag(tag *Tag) error
	DeleteTag(tagID string) error
	TagByUserID(userID int64) (*Tag, error)
}

func NewFromEnv(env string) (Store, error) {
	switch env {
	case "prod":
		return newSQLiteStore("./app.db")
	case "dev":
		return newSQLiteStore("./dev.db")
	default:
		return newSQLiteStore("./dev.db")
	}
}

func New(dbPath string) (Store, error) {
	return newSQLiteStore(dbPath)
}
