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
	// GenerateReceiptPDF membuat PDF "Bukti Donasi" untuk donasi PAID (unduh
	// langsung dari halaman bukti donasi) — menolak (apperror.NotFound) bila
	// donasi belum PAID, sama seperti guard di halaman web-nya.
	GenerateReceiptPDF(ctx context.Context, publicRef string) (data []byte, filename string, err error)
	Status(ctx context.Context, publicRef string) (donation_dto.StatusResponse, error)
	// PublicRecentDonations mengembalikan donasi PAID terbaru untuk campaign
	// PUBLISHED (identifikasi via slug) — dipakai daftar "donatur terbaru" di
	// halaman detail campaign publik, nama donor sudah dimasking bila anonim.
	PublicRecentDonations(ctx context.Context, slug string, limit int) ([]donation_dto.PublicDonationItem, error)
	MyList(ctx context.Context, donorUserID int64, q dto.ListQuery) ([]donation_dto.Response, int, error)
	CMSList(ctx context.Context, q dto.ListQuery, campaignID int64, status string) ([]donation_dto.Response, int, error)
	// CMSGet mengembalikan detail donasi termasuk PII donor — dipakai
	// membuka form edit donasi manual di CMS (lihat donation_dto.AdminDetailResponse).
	CMSGet(ctx context.Context, id int64) (donation_dto.AdminDetailResponse, error)

	// AdminCreate/AdminUpdate/AdminDelete adalah CRUD donasi manual/offline
	// (item 1 revision-prompt-2.md) — menangani donasi yang tidak lewat
	// Amdigipay/Bisatopup (mis. tunai/transfer manual). Selalu gateway="manual",
	// TIDAK PERNAH menyentuh tr_wallet_ledger — saldo campaign yang bisa
	// ditarik (withdrawal) tetap murni berasal dari donasi Bisatopup saja.
	AdminCreate(ctx context.Context, req donation_dto.AdminCreateRequest) (donation_dto.Response, error)
	// AdminUpdate/AdminDelete menolak (apperror.Forbidden) bila donasi target
	// bukan gateway="manual" — donasi Bisatopup adalah catatan finansial yang
	// tidak boleh diubah/dihapus dari sini (pola sama celengan syahid
	// destroyAdminDonation).
	AdminUpdate(ctx context.Context, id int64, req donation_dto.AdminUpdateRequest) (donation_dto.Response, error)
	AdminDelete(ctx context.Context, id int64) error

	// ProcessCallback menangani webhook payment callback Bisabiller: verifikasi
	// signature, idempotency (status final tidak pernah ditimpa ulang kecuali
	// EXPIRED yang di-override PAID oleh late callback), row lock, dan validasi
	// amount sebelum status donasi diperbarui dan (bila PAID) saldo campaign
	// dikreditkan lewat wallet_service dalam transaksi yang sama.
	ProcessCallback(ctx context.Context, req donation_dto.CallbackRequest) error

	// ExpireStale menandai EXPIRED seluruh donasi PENDING yang sudah lewat
	// expiredDate. Dipanggil langsung oleh RunExpireScheduler tiap interval
	// tetap (§9.6/§13.4 techspec — job terjadwal `donation.expire_check`).
	ExpireStale(ctx context.Context) (int64, error)

	// RunExpireScheduler menjalankan ExpireStale secara periodik (goroutine
	// time.Ticker, bukan lewat job queue — §13.4 techspec) sampai proses
	// berhenti. Dipanggil sekali sebagai `go donationSvc.RunExpireScheduler()`
	// dari router.go.
	RunExpireScheduler()
}
