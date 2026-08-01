package permission_repository

import (
	"context"

	"fsldk-api/modules/permission/permission_model"

	"gorm.io/gorm"
)

// RepositoryImpl adalah implementasi Repository berbasis GORM.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) ListAll(ctx context.Context) ([]permission_model.Permission, error) {
	var out []permission_model.Permission
	err := r.db.WithContext(ctx).Table("lk_permission").
		Where("isActive = 1").
		Order("moduleName, permissionCode").
		Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) CodesByRole(ctx context.Context, roleID int64) ([]string, error) {
	var out []string
	err := r.db.WithContext(ctx).Table("lk_permission p").
		Select("p.permissionCode").
		Joins("JOIN map_role_permission mrp ON mrp.permissionID = p.permissionID").
		Where("mrp.roleID = ? AND p.isActive = 1", roleID).
		Scan(&out).Error
	return out, err
}

func (r *RepositoryImpl) MenuByRole(ctx context.Context, roleID int64) ([]permission_model.MenuItem, error) {
	var out []permission_model.MenuItem
	err := r.db.WithContext(ctx).Table("lk_permission p").
		Select("p.menuLabel, p.menuIcon, p.menuRoute, p.sortOrder").
		Joins("JOIN map_role_permission mrp ON mrp.permissionID = p.permissionID").
		Where("mrp.roleID = ? AND p.isActive = 1 AND p.menuRoute IS NOT NULL AND p.menuRoute <> ''", roleID).
		Order("p.sortOrder ASC").
		Scan(&out).Error
	return out, err
}
