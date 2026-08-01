// Package role_service memuat logika bisnis modul role.
package role_service

import (
	"context"

	"fsldk-api/modules/role/role_dto"
	"fsldk-api/modules/role/role_model"
)

// Service adalah kontrak logika bisnis role.
type Service interface {
	List(ctx context.Context, search string) ([]role_dto.Response, error)
	Get(ctx context.Context, id int64) (role_dto.Response, error)
	Create(ctx context.Context, req role_dto.CreateRequest) (role_dto.Response, error)
	Update(ctx context.Context, id int64, req role_dto.UpdateRequest) (role_dto.Response, error)
	Delete(ctx context.Context, id int64) error
	SetPermissions(ctx context.Context, id int64, ids []int64) (role_dto.Response, error)
	Users(ctx context.Context, id int64) ([]role_model.RoleUser, error)
}
