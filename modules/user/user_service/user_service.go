// Package user_service memuat logika bisnis modul user.
package user_service

import (
	"context"

	"fsldk-api/base/dto"
	"fsldk-api/modules/user/user_dto"
)

// Service adalah kontrak logika bisnis pengguna.
type Service interface {
	List(ctx context.Context, q dto.ListQuery, roleID int64) ([]user_dto.Response, int, error)
	Get(ctx context.Context, id int64) (user_dto.Response, error)
	Create(ctx context.Context, req user_dto.CreateRequest, actorID int64) (user_dto.Response, error)
	Update(ctx context.Context, id int64, req user_dto.UpdateRequest, actorID int64) (user_dto.Response, error)
	SetStatus(ctx context.Context, id int64, active bool, actorID int64) error
	ResetPassword(ctx context.Context, id int64) (string, error)
	Delete(ctx context.Context, id, actorID int64) error
}
