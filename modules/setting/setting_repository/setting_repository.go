// Package setting_repository adalah lapisan akses data modul setting (GORM).
package setting_repository

import (
	"context"
	"errors"

	"fsldk-api/modules/setting/setting_model"
)

// ErrNotFound dikembalikan bila setting tidak ditemukan.
var ErrNotFound = errors.New("setting tidak ditemukan")

// Repository adalah kontrak akses data setting.
type Repository interface {
	List(ctx context.Context) ([]setting_model.Setting, error)
	FindByID(ctx context.Context, id int64) (setting_model.Setting, error)
	FindByGroupKey(ctx context.Context, group, key string) (setting_model.Setting, error)
	Update(ctx context.Context, id int64, value string, updatedBy int64) error
}
