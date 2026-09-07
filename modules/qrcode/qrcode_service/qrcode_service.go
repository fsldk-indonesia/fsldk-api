// Package qrcode_service memuat logika bisnis modul QR Code.
package qrcode_service

import (
	"context"

	"fsldk-api/base/dto"
	"fsldk-api/modules/qrcode/qrcode_dto"
)

// Service adalah kontrak logika bisnis QR Code.
type Service interface {
	List(ctx context.Context, q dto.ListQuery) ([]qrcode_dto.Response, int, error)
	Get(ctx context.Context, id int64) (qrcode_dto.Response, error)
	Create(ctx context.Context, req qrcode_dto.CreateRequest, actorID int64) (qrcode_dto.Response, error)
	Update(ctx context.Context, id int64, req qrcode_dto.UpdateRequest, actorID int64) (qrcode_dto.Response, error)
	Delete(ctx context.Context, id int64) error
	// Image mengembalikan byte PNG QR untuk sebuah baris — meng-encode
	// destinationURL langsung dan menerapkan warna/ikon-tengah/caption yang
	// tersimpan. size dibatasi pemanggil (handler).
	Image(ctx context.Context, id int64, size int) ([]byte, error)
}
