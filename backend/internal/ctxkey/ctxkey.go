package ctxkey

type Key string

const (
	UserID  Key = "user_id"
	Email   Key = "email"
	IsAdmin Key = "is_admin"
	JTI     Key = "jti"
)
