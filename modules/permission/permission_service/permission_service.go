// Package permission_service memuat logika bisnis modul permission & menu.
package permission_service

import (
	"context"

	"fsldk-api/modules/permission/permission_model"
)

// Service menyediakan logika permission & menu, sekaligus memenuhi kontrak
// middlewares.PermissionLoader melalui method RolePermissions.
type Service interface {
	RolePermissions(ctx context.Context, roleID int64) ([]string, error)
	Menu(ctx context.Context, roleID int64) ([]permission_model.MenuItem, error)
	ListAll(ctx context.Context) ([]permission_model.Permission, error)
}
