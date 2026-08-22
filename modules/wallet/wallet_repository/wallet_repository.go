// Package wallet_repository menyediakan akses data modul wallet.
package wallet_repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"fsldk-api/modules/wallet/wallet_dto"
	"fsldk-api/modules/wallet/wallet_model"
)

// ErrDuplicateLedgerEntry menandakan insert ledger ditolak oleh
// UNIQUE(referenceType, referenceID, entryType) — event yang sama sudah
// pernah dicatat sebelumnya. Ini guard idempotency di level DB.
var ErrDuplicateLedgerEntry = errors.New("ledger entry sudah pernah dicatat untuk referensi ini")

// Repository adalah kontrak akses data tr_wallet_ledger. InsertLedgerEntry
// adalah satu-satunya cara menulis ke ledger — dipanggil lewat
// wallet_service, tidak pernah langsung dari modul lain.
type Repository interface {
	// InsertLedgerEntry mengunci baris ms_campaign, menghitung balanceAfter
	// baru dari baris ledger terakhir, lalu insert satu baris ledger di
	// dalam tx yang disediakan caller. Mengembalikan ErrDuplicateLedgerEntry
	// bila (referenceType, referenceID, entryType) sudah pernah dicatat.
	InsertLedgerEntry(tx *gorm.DB, p wallet_model.LedgerEntryParams) (ledgerID int64, balanceAfter float64, err error)

	// LatestBalance mengembalikan balanceAfter baris ledger terakhir untuk
	// campaign tsb, atau 0 bila belum ada baris ledger sama sekali.
	LatestBalance(ctx context.Context, campaignID int64) (float64, error)

	// LatestBalanceForUpdate mengunci baris ms_campaign lalu mengembalikan
	// balanceAfter baris ledger terakhir — dipakai wallet_service untuk
	// validasi saldo (mis. ReserveWithdrawal) sebelum InsertLedgerEntry,
	// di dalam tx yang sama (lock InnoDB bersifat reentrant per-transaksi,
	// aman dipanggil lagi oleh InsertLedgerEntry setelahnya).
	LatestBalanceForUpdate(tx *gorm.DB, campaignID int64) (float64, error)

	// SumByEntryType menjumlahkan amount seluruh baris ledger dengan
	// entryType tertentu untuk satu campaign.
	SumByEntryType(ctx context.Context, campaignID int64, entryType string) (float64, error)

	// SumPendingReserve menjumlahkan (WITHDRAWAL_RESERVE - WITHDRAWAL_RELEASE)
	// untuk withdrawal yang statusnya belum final (definisi pendingBalance).
	SumPendingReserve(ctx context.Context, campaignID int64) (float64, error)

	// SumWithdrawnSuccess menjumlahkan tr_withdrawal.amount untuk withdrawal
	// berstatus SUCCESS (bukan dari ledger — saat SUCCESS saldo sudah didebit
	// sejak reservasi, tidak ada entry ledger baru).
	SumWithdrawnSuccess(ctx context.Context, campaignID int64) (float64, error)

	// ListLedger mengembalikan riwayat ledger satu campaign, terpaginasi.
	ListLedger(ctx context.Context, campaignID int64, filter wallet_dto.LedgerListFilter) ([]wallet_model.LedgerEntry, int64, error)
}
