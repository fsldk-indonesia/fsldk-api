// Package user_repository adalah lapisan akses data modul user (GORM).
package user_repository

import (
	"context"
	"errors"

	"fsldk-api/modules/user/user_dto"
	"fsldk-api/modules/user/user_model"
)

// ErrNotFound dikembalikan bila pengguna tidak ditemukan.
var ErrNotFound = errors.New("user tidak ditemukan")

// Repository adalah kontrak akses data pengguna.
type Repository interface {
	FindByID(ctx context.Context, id int64) (user_model.User, error)
	FindByEmail(ctx context.Context, email string) (user_model.User, error)
	FindByGoogleID(ctx context.Context, googleID string) (user_model.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	Create(ctx context.Context, p user_model.CreateParams) (int64, error)
	List(ctx context.Context, f user_dto.ListFilter) ([]user_model.User, int64, error)
	Update(ctx context.Context, id int64, fullName string, roleID int64, isActive bool, updatedBy int64) error
	SetActive(ctx context.Context, id int64, active bool, updatedBy int64) error
	SetPassword(ctx context.Context, id int64, hashed string, mustChange bool) error
	LinkGoogle(ctx context.Context, id int64, googleID string, markVerified bool) error
	MarkEmailVerified(ctx context.Context, id int64) error
	SoftDelete(ctx context.Context, id int64, updatedBy int64) error
	LogLogin(ctx context.Context, userID int64, ip, ua, status string) error
}
