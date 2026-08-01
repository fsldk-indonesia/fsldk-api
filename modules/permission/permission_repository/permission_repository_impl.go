package permission_repository

import (
	"context"

	"fsldk-api/modules/permission/permission_model"

	"github.com/jmoiron/sqlx"
)

// RepositoryImpl adalah implementasi Repository berbasis sqlx.
type RepositoryImpl struct{ db *sqlx.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *sqlx.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) ListAll(ctx context.Context) ([]permission_model.Permission, error) {
	var out []permission_model.Permission
	err := r.db.SelectContext(ctx, &out,
		`SELECT permissionID, permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder, isActive
		 FROM lk_permission WHERE isActive = 1 ORDER BY moduleName, permissionCode`)
	return out, err
}

func (r *RepositoryImpl) CodesByRole(ctx context.Context, roleID int64) ([]string, error) {
	var out []string
	err := r.db.SelectContext(ctx, &out,
		`SELECT p.permissionCode
		 FROM lk_permission p
		 JOIN map_role_permission mrp ON mrp.permissionID = p.permissionID
		 WHERE mrp.roleID = ? AND p.isActive = 1`, roleID)
	return out, err
}

func (r *RepositoryImpl) MenuByRole(ctx context.Context, roleID int64) ([]permission_model.MenuItem, error) {
	var out []permission_model.MenuItem
	err := r.db.SelectContext(ctx, &out,
		`SELECT p.menuLabel, p.menuIcon, p.menuRoute, p.sortOrder
		 FROM lk_permission p
		 JOIN map_role_permission mrp ON mrp.permissionID = p.permissionID
		 WHERE mrp.roleID = ? AND p.isActive = 1
		   AND p.menuRoute IS NOT NULL AND p.menuRoute <> ''
		 ORDER BY p.sortOrder ASC`, roleID)
	return out, err
}
