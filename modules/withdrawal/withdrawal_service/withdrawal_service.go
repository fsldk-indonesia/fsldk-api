// Package withdrawal_service memuat logika bisnis modul withdrawal.
package withdrawal_service

import (
	"context"

	"fsldk-api/base/dto"
	"fsldk-api/modules/withdrawal/withdrawal_dto"
)

// Service adalah kontrak logika bisnis withdrawal.
type Service interface {
	// Request mengajukan penarikan saldo baru — murni permission-gated
	// (kantong_amal.withdrawal.request), TIDAK ada lagi kepemilikan campaign
	// (revisi 2026-09-01). Rekening tujuan diinput ulang tiap pengajuan
	// (req.BeneficiaryBankCode/AccountNumber), divalidasi live via inquiry
	// gateway sebelum direservasi. Status awal SECURITY_CHECK (lihat
	// withdrawal_service_impl.go).
	Request(ctx context.Context, campaignID, requesterUserID int64, req withdrawal_dto.CreateRequest) (withdrawal_dto.Response, error)
	CMSList(ctx context.Context, q dto.ListQuery, status string) ([]withdrawal_dto.Response, int, error)
	// Detail mengembalikan satu withdrawal untuk halaman detail CMS (timeline
	// status) — permission-gated sama dengan CMSList (PermWithdrawalApprove),
	// bukan lagi harus pengaju asli.
	Detail(ctx context.Context, withdrawalID int64) (withdrawal_dto.Response, error)
	// Cancel hanya berlaku saat status REQUESTED, SECURITY_CHECK, atau
	// APPROVED (sebelum diproses) — permission-gated, bukan lagi harus
	// pengaju asli (revisi 2026-09-01).
	Cancel(ctx context.Context, withdrawalID, requesterUserID int64) error

	// RequestSecurityOtp membuat OTP challenge baru untuk withdrawal yang
	// sedang SECURITY_CHECK dan mengirim kodenya via WhatsApp lewat job queue
	// (§14.4 techspec — tidak pernah sinkron/langsung ke Kirimdev).
	RequestSecurityOtp(ctx context.Context, withdrawalID, requesterUserID int64) error
	// VerifySecurity menyelesaikan step SECURITY_CHECK → APPROVED (siap
	// diproses). Withdrawal rutin (nominal wajar, bukan withdrawal pertama
	// campaign) cukup Password (Option B); trigger risk-based (nominal
	// >Rp10 juta atau withdrawal pertama campaign) mewajibkan OtpCode juga
	// (Option D). Revisi (2026-08-30): tidak ada lagi langkah approve
	// terpisah oleh orang lain (maker-checker dihapus) — begitu verifikasi
	// ini lolos, withdrawal langsung siap diproses lewat Process().
	VerifySecurity(ctx context.Context, withdrawalID, requesterUserID int64, req withdrawal_dto.SecurityVerifyRequest) (withdrawal_dto.Response, error)
	// Process memicu disbursement ke gateway — permission-gated
	// (kantong_amal.withdrawal.process), tidak lagi mensyaratkan aktor
	// berbeda dari requester. Sukses/timeout keduanya berujung PROCESSING
	// (final status ditentukan callback async); hanya penolakan eksplisit
	// gateway yang langsung FAILED + release reservasi.
	Process(ctx context.Context, withdrawalID, actorUserID int64) (withdrawal_dto.Response, error)
	// ProcessCallback menangani webhook disbursement — diamankan URL path
	// secret (bukan signature, Bisabiller tidak mengirimkannya untuk transfer).
	ProcessCallback(ctx context.Context, req withdrawal_dto.DisbursementCallbackRequest, urlSecret string) error

	Inquiry(ctx context.Context, req withdrawal_dto.InquiryRequest) (withdrawal_dto.InquiryResponse, error)
	ListBanks(ctx context.Context) ([]withdrawal_dto.BankListItem, error)

	// ReconcileStaleProcessing mengembalikan withdrawal PROCESSING yang
	// sudah melewati threshold waktu wajar — kandidat tinjauan manual admin.
	// Tidak ada auto-fix (§11.7 — status final hanya lewat callback gateway
	// atau tindakan admin eksplisit); dipanggil periodik oleh
	// RunReconcileScheduler untuk mencatatnya ke log operasional.
	ReconcileStaleProcessing(ctx context.Context) ([]withdrawal_dto.Response, error)

	// RunReconcileScheduler menjalankan ReconcileStaleProcessing secara
	// periodik (goroutine time.Ticker, bukan lewat job queue — §13.4
	// techspec, job `withdrawal.reconcile_check`) sampai proses berhenti.
	// Dipanggil sekali sebagai `go withdrawalSvc.RunReconcileScheduler()`
	// dari router.go.
	RunReconcileScheduler()
}
