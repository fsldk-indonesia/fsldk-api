// Package permission_model memuat entitas modul permission. Murni struct data.
package permission_model

import "database/sql"

// Permission merepresentasikan satu baris lk_permission.
type Permission struct {
	PermissionID   int64          `gorm:"column:permissionID;primaryKey" json:"permissionID"`
	PermissionCode string         `gorm:"column:permissionCode" json:"permissionCode"`
	PermissionName string         `gorm:"column:permissionName" json:"permissionName"`
	ModuleName     string         `gorm:"column:moduleName" json:"moduleName"`
	MenuLabel      sql.NullString `gorm:"column:menuLabel" json:"-"`
	MenuIcon       sql.NullString `gorm:"column:menuIcon" json:"-"`
	MenuRoute      sql.NullString `gorm:"column:menuRoute" json:"-"`
	SortOrder      sql.NullInt64  `gorm:"column:sortOrder" json:"-"`
	IsActive       bool           `gorm:"column:isActive" json:"isActive"`
}

// MenuItem adalah item menu sidebar CMS.
type MenuItem struct {
	MenuLabel string `gorm:"column:menuLabel" json:"menuLabel"`
	MenuIcon  string `gorm:"column:menuIcon" json:"menuIcon"`
	MenuRoute string `gorm:"column:menuRoute" json:"menuRoute"`
	SortOrder int    `gorm:"column:sortOrder" json:"sortOrder"`
}
