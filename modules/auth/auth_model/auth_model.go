// Package auth_model memuat entitas modul auth. Murni struct data.
package auth_model

import (
	"database/sql"
	"time"
)

// EmailToken merepresentasikan token verifikasi email / reset password.
type EmailToken struct {
	TokenID   int64        `gorm:"column:tokenID;primaryKey"`
	UserID    int64        `gorm:"column:userID"`
	Token     string       `gorm:"column:token"`
	TokenType string       `gorm:"column:tokenType"`
	ExpiresAt time.Time    `gorm:"column:expiresAt"`
	UsedAt    sql.NullTime `gorm:"column:usedAt"`
}
