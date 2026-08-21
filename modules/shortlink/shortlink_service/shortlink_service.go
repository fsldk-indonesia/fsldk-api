// Package shortlink_service memuat logika bisnis modul shortlink.
package shortlink_service

import (
	"context"

	"fsldk-api/base/dto"
	"fsldk-api/modules/shortlink/shortlink_dto"
)

// Service adalah kontrak logika bisnis shortlink.
type Service interface {
	List(ctx context.Context, q dto.ListQuery) ([]shortlink_dto.Response, int, error)
	Get(ctx context.Context, id int64) (shortlink_dto.Response, error)
	Create(ctx context.Context, req shortlink_dto.CreateRequest, actorID int64) (shortlink_dto.Response, error)
	Update(ctx context.Context, id int64, req shortlink_dto.UpdateRequest, actorID int64) (shortlink_dto.Response, error)
	Delete(ctx context.Context, id int64) error
	// Resolve mencari shortlink berdasarkan kunci, mencatat kunjungan, dan
	// mengembalikan URL tujuan untuk di-redirect oleh handler.
	Resolve(ctx context.Context, key string) (string, error)

	// GenerateUniqueKey & KeyExists dipakai ulang oleh shortlinkrequest_service
	// saat approve (§6 techspec) — murni baca, tidak menulis, supaya logic
	// generate-key acak yang sudah ada di Create tidak diduplikasi.
	GenerateUniqueKey(ctx context.Context) (string, error)
	KeyExists(ctx context.Context, key string) (bool, error)
}
