package store

type Store interface {
	CreateUser(user *User) (userId int64, err error)
	GetUserByGoogleId(googleId string) (user *User, err error)
	DeleteUser(userId int64) (err error)
	CreateSession(sessionId string, userId int64, expiresAt int64) (session *Session, err error)
	DeleteSessionByUserId(userId int64) (err error)
	DeleteSessionBySessionId(sessionId string) (err error)
	GetSessionAndUserBySessionId(sessionId string) (session *Session, user *User, err error)
	RefreshSession(sessionId string, newExpiresAt int64) (err error)
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
