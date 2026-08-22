// Package donation_repository adalah lapisan akses data modul donation (GORM).
package donation_repository

import (
	"context"
	"errors"

	"fsldk-api/modules/donation/donation_dto"
	"fsldk-api/modules/donation/donation_model"
)

// ErrNotFound dikembalikan bila donasi tidak ditemukan.
var ErrNotFound = errors.New("donasi tidak ditemukan")

// ErrDuplicateIdempotencyKey dikembalikan bila insert ditolak oleh
// UNIQUE(idempotencyKey) — request dengan key yang sama sudah pernah
// diproses sebelumnya.
var ErrDuplicateIdempotencyKey = errors.New("idempotencyKey sudah pernah dipakai")

// Repository adalah kontrak akses data donation.
type Repository interface {
	Create(ctx context.Context, p donation_model.CreateParams) (int64, error)
	FindByID(ctx context.Context, id int64) (donation_model.Donation, error)
	FindByPublicRef(ctx context.Context, publicRef string) (donation_model.Donation, error)
	FindByIdempotencyKey(ctx context.Context, key string) (donation_model.Donation, error)
	List(ctx context.Context, f donation_dto.ListFilter) ([]donation_model.Donation, int64, error)
}
