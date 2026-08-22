// Package donation_service memuat logika bisnis modul donation.
package donation_service

import (
	"context"

	"fsldk-api/base/dto"
	"fsldk-api/modules/donation/donation_dto"
)

// Service adalah kontrak logika bisnis donation.
type Service interface {
	// Create membuat donasi baru ke campaign (identifikasi via slug) yang
	// harus berstatus PUBLISHED dan belum lewat deadline, lalu langsung
	// membuat transaksi QRIS ke payment gateway. donorUserID nil berarti
	// donasi tamu (guest, tidak login).
	Create(ctx context.Context, slug string, donorUserID *int64, req donation_dto.CreateRequest) (donation_dto.Response, error)
	GetByPublicRef(ctx context.Context, publicRef string) (donation_dto.Response, error)
	Status(ctx context.Context, publicRef string) (donation_dto.StatusResponse, error)
	MyList(ctx context.Context, donorUserID int64, q dto.ListQuery) ([]donation_dto.Response, int, error)
	CMSList(ctx context.Context, q dto.ListQuery, campaignID int64, status string) ([]donation_dto.Response, int, error)

	// ProcessCallback menangani webhook payment callback Bisabiller: verifikasi
	// signature, idempotency (status final tidak pernah ditimpa ulang kecuali
	// EXPIRED yang di-override PAID oleh late callback), row lock, dan validasi
	// amount sebelum status donasi diperbarui.
	ProcessCallback(ctx context.Context, req donation_dto.CallbackRequest) error
}
