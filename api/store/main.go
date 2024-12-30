package store

type Store interface {
	CreateUser(user *User) (userId int64, err error)
	GetUserByGoogleId(googleId string) (user *User, err error)
	DeleteUser(userId int64) (err error)
	CreateSession(sessionId int64, userId int64, expiresAt int64) (session *Session, err error)
	DeleteSessionByUserId(userId int64) (err error)
	DeleteSessionBySessionId(sessionId int64) (err error)
	GetSessionAndUserBySessionId(sessionId int64) (session *Session, user *User, err error)
	RefreshSession(sessionId int64, newExpiresAt int64) (err error)
}

func New(env string) (Store, error) {
	switch env {
	case "prod":
		return newSQLiteStore("./data/app.db")
	case "dev":
		return newSQLiteStore("./dev.db")
	default:
		return newSQLiteStore("./dev.db")
	}
}
