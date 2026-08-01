// Package role_model memuat entitas modul role.
package role_model

import (
	"database/sql"
	"time"
)

// Role merepresentasikan satu baris ms_role.
type Role struct {
	RoleID          int64          `db:"roleID" json:"roleID"`
	RoleName        string         `db:"roleName" json:"roleName"`
	RoleDescription sql.NullString `db:"roleDescription" json:"roleDescription"`
	IsSystemRole    bool           `db:"isSystemRole" json:"isSystemRole"`
	IsActive        bool           `db:"isActive" json:"isActive"`
	CreatedDate     time.Time      `db:"createdDate" json:"createdDate"`
	UserCount       int            `db:"userCount" json:"userCount"`
}

// RoleUser adalah ringkasan pengguna pemilik sebuah role.
type RoleUser struct {
	UserID   int64  `db:"userID" json:"userID"`
	FullName string `db:"fullName" json:"fullName"`
	Email    string `db:"email" json:"email"`
	IsActive bool   `db:"isActive" json:"isActive"`
}
