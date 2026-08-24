// Package withdrawal_service memuat logika bisnis modul withdrawal.
package withdrawal_service

import (
	"context"

	"fsldk-api/base/dto"
	"fsldk-api/modules/withdrawal/withdrawal_dto"
)

// Service adalah kontrak logika bisnis withdrawal.
type Service interface {
	// Request mengajukan penarikan saldo baru — beneficiary selalu memakai
	// rekening default campaign, divalidasi ulang via inquiry live sebelum
	// direservasi. Status awal SECURITY_CHECK (lihat withdrawal_service_impl.go).
	Request(ctx context.Context, campaignID, requesterUserID int64, req withdrawal_dto.CreateRequest) (withdrawal_dto.Response, error)
	MyList(ctx context.Context, requesterUserID int64, q dto.ListQuery) ([]withdrawal_dto.Response, int, error)
	CMSList(ctx context.Context, q dto.ListQuery, status string) ([]withdrawal_dto.Response, int, error)
	// Cancel dibatalkan pengaju sendiri — hanya saat status REQUESTED,
	// SECURITY_CHECK, atau PENDING_APPROVAL.
	Cancel(ctx context.Context, withdrawalID, requesterUserID int64) error

	// RequestSecurityOtp membuat OTP challenge baru untuk withdrawal yang
	// sedang SECURITY_CHECK — kode "dikirim" dengan dicatat ke log (dev-mode,
	// pengiriman WhatsApp sungguhan menunggu queue Phase 8, lihat komentar
	// implementasi).
	RequestSecurityOtp(ctx context.Context, withdrawalID, requesterUserID int64) error
	// VerifySecurity menyelesaikan step SECURITY_CHECK → PENDING_APPROVAL.
	// Withdrawal rutin (nominal wajar, bukan withdrawal pertama campaign)
	// cukup Password (Option B); trigger risk-based (nominal >Rp10 juta atau
	// withdrawal pertama campaign) mewajibkan OtpCode juga (Option D).
	VerifySecurity(ctx context.Context, withdrawalID, requesterUserID int64, req withdrawal_dto.SecurityVerifyRequest) (withdrawal_dto.Response, error)
	// Approve mensyaratkan approverUserID berbeda dari requestedByUserID
	// (maker-checker, ADR-006/OQ-04).
	Approve(ctx context.Context, withdrawalID, approverUserID int64) (withdrawal_dto.Response, error)
	Reject(ctx context.Context, withdrawalID, approverUserID int64, reason string) error
	// Process memicu disbursement ke gateway — sukses/timeout keduanya
	// berujung PROCESSING (final status ditentukan callback async); hanya
	// penolakan eksplisit gateway yang langsung FAILED + release reservasi.
	Process(ctx context.Context, withdrawalID, actorUserID int64) (withdrawal_dto.Response, error)
	// ProcessCallback menangani webhook disbursement — diamankan URL path
	// secret (bukan signature, Bisabiller tidak mengirimkannya untuk transfer).
	ProcessCallback(ctx context.Context, req withdrawal_dto.DisbursementCallbackRequest, urlSecret string) error

	Inquiry(ctx context.Context, req withdrawal_dto.InquiryRequest) (withdrawal_dto.InquiryResponse, error)
	ListBanks(ctx context.Context) ([]withdrawal_dto.BankListItem, error)

	// ReconcileStaleProcessing mengembalikan withdrawal PROCESSING yang
	// sudah melewati threshold waktu wajar — kandidat tinjauan manual admin.
	// Belum disambungkan ke scheduler apa pun (Phase 8).
	ReconcileStaleProcessing(ctx context.Context) ([]withdrawal_dto.Response, error)
}
