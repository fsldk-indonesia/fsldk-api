// Package role_repository adalah lapisan akses data modul role.
package role_repository

import (
	"context"
	"errors"

	"fsldk-api/modules/role/role_model"
)

// ErrNotFound dikembalikan bila role tidak ditemukan.
var ErrNotFound = errors.New("role tidak ditemukan")

// Repository adalah kontrak akses data role.
type Repository interface {
	IDByName(ctx context.Context, name string) (int64, error)
	FindByID(ctx context.Context, id int64) (role_model.Role, error)
	ListWithUserCount(ctx context.Context, search string) ([]role_model.Role, error)
	Create(ctx context.Context, name, desc string) (int64, error)
	Update(ctx context.Context, id int64, name, desc string, isActive bool) error
	Delete(ctx context.Context, id int64) error
	CountUsers(ctx context.Context, id int64) (int, error)
	PermissionIDs(ctx context.Context, roleID int64) ([]int64, error)
	PermissionNames(ctx context.Context, roleID int64) ([]string, error)
	SetPermissions(ctx context.Context, roleID int64, permissionIDs []int64) error
	UsersByRole(ctx context.Context, roleID int64) ([]role_model.RoleUser, error)
}
