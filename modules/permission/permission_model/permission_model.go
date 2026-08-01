// Package permission_model memuat entitas modul permission.
package permission_model

import "database/sql"

// Permission merepresentasikan satu baris lk_permission.
type Permission struct {
	PermissionID   int64          `db:"permissionID" json:"permissionID"`
	PermissionCode string         `db:"permissionCode" json:"permissionCode"`
	PermissionName string         `db:"permissionName" json:"permissionName"`
	ModuleName     string         `db:"moduleName" json:"moduleName"`
	MenuLabel      sql.NullString `db:"menuLabel" json:"-"`
	MenuIcon       sql.NullString `db:"menuIcon" json:"-"`
	MenuRoute      sql.NullString `db:"menuRoute" json:"-"`
	SortOrder      sql.NullInt64  `db:"sortOrder" json:"-"`
	IsActive       bool           `db:"isActive" json:"isActive"`
}

// MenuItem adalah item menu sidebar CMS.
type MenuItem struct {
	MenuLabel string `db:"menuLabel" json:"menuLabel"`
	MenuIcon  string `db:"menuIcon" json:"menuIcon"`
	MenuRoute string `db:"menuRoute" json:"menuRoute"`
	SortOrder int    `db:"sortOrder" json:"sortOrder"`
}
