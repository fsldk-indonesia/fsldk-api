// Package auth_model memuat entitas modul auth.
package auth_model

import (
	"database/sql"
	"time"
)

// EmailToken merepresentasikan token verifikasi email / reset password.
type EmailToken struct {
	TokenID   int64        `db:"tokenID"`
	UserID    int64        `db:"userID"`
	Token     string       `db:"token"`
	TokenType string       `db:"tokenType"`
	ExpiresAt time.Time    `db:"expiresAt"`
	UsedAt    sql.NullTime `db:"usedAt"`
}
