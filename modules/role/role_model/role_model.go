// Package role_model memuat entitas modul role. Murni struct data.
package role_model

import (
	"database/sql"
	"time"
)

// Role merepresentasikan satu baris ms_role. UserCount bersifat read-only
// (hasil subquery), ditandai `->` agar tidak ikut tertulis saat Create/Update.
type Role struct {
	RoleID          int64          `gorm:"column:roleID;primaryKey" json:"roleID"`
	RoleName        string         `gorm:"column:roleName" json:"roleName"`
	RoleDescription sql.NullString `gorm:"column:roleDescription" json:"roleDescription"`
	IsSystemRole    bool           `gorm:"column:isSystemRole" json:"isSystemRole"`
	IsActive        bool           `gorm:"column:isActive" json:"isActive"`
	CreatedDate     time.Time      `gorm:"column:createdDate" json:"createdDate"`
	UserCount       int            `gorm:"column:userCount;->" json:"userCount"`
}

// RoleUser adalah ringkasan pengguna pemilik sebuah role.
type RoleUser struct {
	UserID   int64  `gorm:"column:userID" json:"userID"`
	FullName string `gorm:"column:fullName" json:"fullName"`
	Email    string `gorm:"column:email" json:"email"`
	IsActive bool   `gorm:"column:isActive" json:"isActive"`
}
