// Package permission_repository adalah lapisan akses data modul permission.
package permission_repository

import (
	"context"

	"fsldk-api/modules/permission/permission_model"
)

// Repository adalah kontrak akses data permission.
type Repository interface {
	ListAll(ctx context.Context) ([]permission_model.Permission, error)
	CodesByRole(ctx context.Context, roleID int64) ([]string, error)
	MenuByRole(ctx context.Context, roleID int64) ([]permission_model.MenuItem, error)
}
