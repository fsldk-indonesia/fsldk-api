// Package shortlink_repository adalah lapisan akses data modul shortlink (GORM).
package shortlink_repository

import (
	"context"
	"errors"

	"fsldk-api/modules/shortlink/shortlink_dto"
	"fsldk-api/modules/shortlink/shortlink_model"
)

// ErrNotFound dikembalikan bila shortlink tidak ditemukan.
var ErrNotFound = errors.New("shortlink tidak ditemukan")

// Repository adalah kontrak akses data shortlink.
type Repository interface {
	FindByID(ctx context.Context, id int64) (shortlink_model.ShortLink, error)
	FindByKey(ctx context.Context, key string) (shortlink_model.ShortLink, error)
	ExistsByKey(ctx context.Context, key string, exceptID int64) (bool, error)
	List(ctx context.Context, f shortlink_dto.ListFilter) ([]shortlink_model.ShortLink, int64, error)
	Create(ctx context.Context, shortKey, destinationURL string, createdBy int64) (int64, error)
	Update(ctx context.Context, id int64, shortKey, destinationURL string, updatedBy int64) error
	IncrementVisit(ctx context.Context, id int64) error
	Delete(ctx context.Context, id int64) error
}
