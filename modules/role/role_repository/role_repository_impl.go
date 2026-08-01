package role_repository

import (
	"context"
	"errors"
	"time"

	"fsldk-api/modules/role/role_model"

	"gorm.io/gorm"
)

// RepositoryImpl adalah implementasi Repository berbasis GORM.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) IDByName(ctx context.Context, name string) (int64, error) {
	var id int64
	err := r.db.WithContext(ctx).Table("ms_role").
		Select("roleID").Where("roleName = ?", name).Take(&id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, ErrNotFound
	}
	return id, err
}

func (r *RepositoryImpl) FindByID(ctx context.Context, id int64) (role_model.Role, error) {
	var role role_model.Role
	err := r.db.WithContext(ctx).Table("ms_role").
		Select("roleID, roleName, roleDescription, isSystemRole, isActive, createdDate, 0 AS userCount").
		Where("roleID = ?", id).Take(&role).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return role_model.Role{}, ErrNotFound
	}
	return role, err
}

func (r *RepositoryImpl) ListWithUserCount(ctx context.Context, search string) ([]role_model.Role, error) {
	q := r.db.WithContext(ctx).Table("ms_role r").
		Select("r.roleID, r.roleName, r.roleDescription, r.isSystemRole, r.isActive, r.createdDate, " +
			"(SELECT COUNT(*) FROM ms_user u WHERE u.roleID = r.roleID) AS userCount")
	if search != "" {
		q = q.Where("r.roleName LIKE ?", "%"+search+"%")
	}
	var out []role_model.Role
	err := q.Order("r.roleID ASC").Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) Create(ctx context.Context, name, desc string) (int64, error) {
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("ms_role").Create(map[string]interface{}{
			"roleName":        name,
			"roleDescription": desc,
			"isSystemRole":    false,
			"isActive":        true,
			"createdDate":     time.Now(),
		}).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) Update(ctx context.Context, id int64, name, desc string, isActive bool) error {
	return r.db.WithContext(ctx).Table("ms_role").Where("roleID = ?", id).Updates(map[string]interface{}{
		"roleName":        name,
		"roleDescription": desc,
		"isActive":        isActive,
		"updatedDate":     time.Now(),
	}).Error
}

func (r *RepositoryImpl) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Exec("DELETE FROM ms_role WHERE roleID = ?", id).Error
}

func (r *RepositoryImpl) CountUsers(ctx context.Context, id int64) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("ms_user").Where("roleID = ?", id).Count(&count).Error
	return int(count), err
}

func (r *RepositoryImpl) PermissionIDs(ctx context.Context, roleID int64) ([]int64, error) {
	var ids []int64
	err := r.db.WithContext(ctx).Table("map_role_permission").
		Select("permissionID").Where("roleID = ?", roleID).Scan(&ids).Error
	return ids, err
}

func (r *RepositoryImpl) PermissionNames(ctx context.Context, roleID int64) ([]string, error) {
	var names []string
	err := r.db.WithContext(ctx).Table("lk_permission p").
		Select("p.permissionName").
		Joins("JOIN map_role_permission mrp ON mrp.permissionID = p.permissionID").
		Where("mrp.roleID = ?", roleID).
		Order("p.moduleName").
		Scan(&names).Error
	return names, err
}

func (r *RepositoryImpl) UsersByRole(ctx context.Context, roleID int64) ([]role_model.RoleUser, error) {
	var out []role_model.RoleUser
	err := r.db.WithContext(ctx).Table("ms_user").
		Select("userID, fullName, email, isActive").
		Where("roleID = ?", roleID).Order("fullName").Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) SetPermissions(ctx context.Context, roleID int64, permissionIDs []int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM map_role_permission WHERE roleID = ?", roleID).Error; err != nil {
			return err
		}
		for _, pid := range permissionIDs {
			if err := tx.Table("map_role_permission").Create(map[string]interface{}{
				"roleID":       roleID,
				"permissionID": pid,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
