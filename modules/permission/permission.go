// Package permission mengelola daftar permission (hak akses) sekaligus menjadi
// sumber menu sidebar CMS (mengadaptasi pola lk_accesscontrol pada fnb-backend).
package permission

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

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

// Repository adalah kontrak akses data permission.
type Repository interface {
	ListAll(ctx context.Context) ([]Permission, error)
	CodesByRole(ctx context.Context, roleID int64) ([]string, error)
	MenuByRole(ctx context.Context, roleID int64) ([]MenuItem, error)
}

type repository struct{ db *sqlx.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *sqlx.DB) Repository { return &repository{db: db} }

func (r *repository) ListAll(ctx context.Context) ([]Permission, error) {
	var out []Permission
	err := r.db.SelectContext(ctx, &out,
		`SELECT permissionID, permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder, isActive
		 FROM lk_permission WHERE isActive = 1 ORDER BY moduleName, permissionCode`)
	return out, err
}

func (r *repository) CodesByRole(ctx context.Context, roleID int64) ([]string, error) {
	var out []string
	err := r.db.SelectContext(ctx, &out,
		`SELECT p.permissionCode
		 FROM lk_permission p
		 JOIN map_role_permission mrp ON mrp.permissionID = p.permissionID
		 WHERE mrp.roleID = ? AND p.isActive = 1`, roleID)
	return out, err
}

func (r *repository) MenuByRole(ctx context.Context, roleID int64) ([]MenuItem, error) {
	var out []MenuItem
	err := r.db.SelectContext(ctx, &out,
		`SELECT p.menuLabel, p.menuIcon, p.menuRoute, p.sortOrder
		 FROM lk_permission p
		 JOIN map_role_permission mrp ON mrp.permissionID = p.permissionID
		 WHERE mrp.roleID = ? AND p.isActive = 1
		   AND p.menuRoute IS NOT NULL AND p.menuRoute <> ''
		 ORDER BY p.sortOrder ASC`, roleID)
	return out, err
}

// Service menyediakan logika permission & menu, sekaligus memenuhi kontrak
// middlewares.PermissionLoader.
type Service struct{ repo Repository }

// NewService membuat Service.
func NewService(repo Repository) *Service { return &Service{repo: repo} }

// RolePermissions memenuhi middlewares.PermissionLoader.
func (s *Service) RolePermissions(ctx context.Context, roleID int64) ([]string, error) {
	return s.repo.CodesByRole(ctx, roleID)
}

// Menu mengembalikan menu sidebar CMS untuk sebuah role.
func (s *Service) Menu(ctx context.Context, roleID int64) ([]MenuItem, error) {
	items, err := s.repo.MenuByRole(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []MenuItem{}
	}
	return items, nil
}

// ListAll mengembalikan seluruh permission (untuk manajemen role).
func (s *Service) ListAll(ctx context.Context) ([]Permission, error) {
	return s.repo.ListAll(ctx)
}
