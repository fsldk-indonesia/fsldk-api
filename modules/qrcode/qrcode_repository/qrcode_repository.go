// Package qrcode_repository adalah lapisan akses data modul QR Code (GORM).
package qrcode_repository

import (
	"context"
	"errors"

	"fsldk-api/modules/qrcode/qrcode_dto"
	"fsldk-api/modules/qrcode/qrcode_model"
)

// ErrNotFound dikembalikan bila QR Code tidak ditemukan.
var ErrNotFound = errors.New("qr code tidak ditemukan")

// Repository adalah kontrak akses data QR Code. Create/Update menerima nilai
// model (qrcode_model.QRCode) dan hanya menulis kolom yang relevan — bentuk ini
// menghindari daftar argumen string panjang tanpa menambah struct baru di
// lapisan logika.
type Repository interface {
	FindByID(ctx context.Context, id int64) (qrcode_model.QRCode, error)
	List(ctx context.Context, f qrcode_dto.ListFilter) ([]qrcode_model.QRCode, int64, error)
	Create(ctx context.Context, m qrcode_model.QRCode) (int64, error)
	Update(ctx context.Context, id int64, m qrcode_model.QRCode) error
	Delete(ctx context.Context, id int64) error
}
