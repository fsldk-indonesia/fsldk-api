// Package role mengelola role dan pemetaan permission-nya.
package role

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
