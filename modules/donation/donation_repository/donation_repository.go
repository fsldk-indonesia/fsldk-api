// Package donation_repository adalah lapisan akses data modul donation (GORM).
package donation_repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

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

	// CountPaidByCampaign/CountPendingByCampaign dipakai guard
	// campaign_service.Delete() — mencegah campaign yang masih punya donasi
	// aktif/belum ditarik terhapus.
	CountPaidByCampaign(ctx context.Context, campaignID int64) (int64, error)
	CountPendingByCampaign(ctx context.Context, campaignID int64) (int64, error)

	// AdminCreate/AdminUpdate/AdminDelete adalah CRUD donasi manual/offline
	// (gateway="manual") oleh admin CMS — TIDAK PERNAH menyentuh
	// tr_wallet_ledger (lihat donation_service_impl.go). AdminDelete/
	// AdminUpdate wajib dipanggil hanya setelah caller memvalidasi baris
	// bergateway "manual" (repository tidak mengecek ini sendiri).
	AdminCreate(ctx context.Context, p donation_model.AdminCreateParams) (int64, error)
	AdminUpdate(ctx context.Context, id int64, p donation_model.AdminUpdateParams) error
	AdminDelete(ctx context.Context, id int64) error

	// UpdateGatewayResult menyimpan hasil CreateQRISTransaction (qrPayload/
	// paymentCode/paymentLink/externalTransactionID) ke donasi yang baru dibuat.
	UpdateGatewayResult(ctx context.Context, donationID int64, p donation_model.GatewayResultParams) error
	// MarkGatewayFailed menandai donasi FAILED saat panggilan create-transaction
	// ke gateway gagal — tidak ada retry otomatis di titik ini (lihat pkg/bisatopup).
	MarkGatewayFailed(ctx context.Context, donationID int64) error

	// FindByExternalTransactionIDForUpdate mengunci baris donasi (SELECT...FOR
	// UPDATE) di dalam tx yang diberikan — wajib dipanggil di dalam transaksi
	// sebelum UpdateCallbackStatus untuk mencegah race antar callback bersamaan.
	FindByExternalTransactionIDForUpdate(tx *gorm.DB, externalTransactionID string) (donation_model.Donation, error)
	// UpdateCallbackStatus memperbarui status donasi dari hasil callback
	// payment gateway di dalam tx yang sama dengan pengunciannya.
	UpdateCallbackStatus(tx *gorm.DB, donationID int64, p donation_model.CallbackUpdateParams) error

	// ExpireStalePending menandai EXPIRED seluruh donasi PENDING yang sudah
	// melewati expiredDate. Mengembalikan jumlah baris yang di-expire beserta
	// hingga 10 ID pertama (untuk audit donation.auto_expired, §16.1 techspec
	// — reuse pola FACT auto_expire_qris ldksyahid-app).
	ExpireStalePending(ctx context.Context) (count int64, sampleIDs []int64, err error)
}
