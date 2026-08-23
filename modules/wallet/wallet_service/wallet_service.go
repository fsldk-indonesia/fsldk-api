// Package wallet_service memuat business logic modul wallet.
// wallet_service adalah satu-satunya write path ke tr_wallet_ledger —
// donation_service/withdrawal_service (fase berikutnya) memanggil method
// di sini, tidak pernah menulis ledger langsung.
package wallet_service

import (
	"context"

	"gorm.io/gorm"

	"fsldk-api/modules/wallet/wallet_dto"
)

type Service interface {
	// CreditDonation mencatat DONATION_CREDIT — dipanggil donation_service
	// di dalam tx yang sama dengan update status donasi jadi PAID, sehingga
	// keduanya commit atomically bersama.
	CreditDonation(tx *gorm.DB, campaignID, donationID int64, amount float64, note string) error

	// ReserveWithdrawal mencatat WITHDRAWAL_RESERVE (debit) setelah
	// memvalidasi amount <= availableBalance di dalam tx yang sama —
	// mengembalikan apperror.InsufficientBalance bila tidak cukup.
	ReserveWithdrawal(tx *gorm.DB, campaignID, withdrawalID int64, amount float64, actorUserID int64, note string) error

	// ReleaseWithdrawal mencatat WITHDRAWAL_RELEASE (credit) — dipanggil
	// di setiap jalur kegagalan withdrawal (FAILED/REJECTED/CANCELLED)
	// untuk melepas reservasi.
	ReleaseWithdrawal(tx *gorm.DB, campaignID, withdrawalID int64, amount float64, note string) error

	// RefundDebit mencatat REFUND_DEBIT setelah memvalidasi saldo cukup —
	// mengembalikan apperror.InsufficientBalance bila tidak (OQ-21: tidak
	// ada dana talangan otomatis, caller wajib eskalasi manual).
	RefundDebit(tx *gorm.DB, campaignID, donationID int64, amount float64, actorUserID int64, note string) error

	// AdjustBalance mencatat ADJUSTMENT_CREDIT/ADJUSTMENT_DEBIT manual,
	// membuka transaksinya sendiri (tidak ada tabel lain yang perlu commit
	// atomically bersamanya). reason wajib tidak kosong (OQ-20).
	AdjustBalance(ctx context.Context, campaignID int64, amount float64, direction string, actorUserID int64, reason string) error

	// GetBalance mengembalikan ringkasan saldo campaign — dipakai CMS
	// (otorisasi lewat permission, bukan kepemilikan).
	GetBalance(ctx context.Context, campaignID int64) (wallet_dto.BalanceResponse, error)

	// ListLedger mengembalikan riwayat ledger campaign, terpaginasi — dipakai CMS.
	ListLedger(ctx context.Context, campaignID int64, filter wallet_dto.LedgerListFilter) ([]wallet_dto.LedgerListItem, int64, error)

	// GetBalanceForOwner mengembalikan saldo campaign untuk endpoint
	// /me — 404 (bukan 403) bila campaign bukan milik ownerUserID, agar
	// tidak membocorkan keberadaan campaign milik user lain (IDOR-safe,
	// pola sama seperti campaign_service.Update).
	GetBalanceForOwner(ctx context.Context, campaignID, ownerUserID int64) (wallet_dto.BalanceResponse, error)

	// ListLedgerForOwner adalah varian ListLedger dengan validasi kepemilikan
	// yang sama seperti GetBalanceForOwner.
	ListLedgerForOwner(ctx context.Context, campaignID, ownerUserID int64, filter wallet_dto.LedgerListFilter) ([]wallet_dto.LedgerListItem, int64, error)
}
