package store

type User struct {
	ID       int64  `json:"id"`
	GoogleID string `json:"google_id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Picture  string `json:"picture"`
}

type Session struct {
	ID        string `json:"id"`
	UserID    int64  `json:"user_id"`
	ExpiresAt int64  `json:"expires_at"`
}
