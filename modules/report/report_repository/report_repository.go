// Package report_repository adalah lapisan akses data modul report.
package report_repository

import (
	"context"
	"time"

	"fsldk-api/base/dto"
	"fsldk-api/modules/report/report_dto"
)

// Repository adalah kontrak akses data laporan.
type Repository interface {
	SubmissionRows(ctx context.Context, formID int64, status string, organizationIDs []int64) ([]report_dto.SubmissionRow, error)

	// ---------- Kantong Amal (Phase 9) — §15 techspec ----------

	// BalanceBefore mengembalikan balanceAfter baris tr_wallet_ledger
	// terakhir sebelum (strictly <) waktu tertentu — opening balance §15.1.
	// campaignID 0 berarti agregat seluruh campaign (SUM balance terakhir
	// tiap campaign, lewat window function).
	BalanceBefore(ctx context.Context, campaignID int64, before time.Time) (float64, error)
	// BalanceAsOf sama seperti BalanceBefore tapi inklusif (<=) — closing
	// balance §15.1.
	BalanceAsOf(ctx context.Context, campaignID int64, asOf time.Time) (float64, error)
	// LedgerSumByType menjumlahkan tr_wallet_ledger.amount untuk satu
	// entryType dalam periode [from, to] — dipakai incoming/refund/adjustment.
	LedgerSumByType(ctx context.Context, campaignID int64, entryType string, from, to time.Time) (float64, error)
	// WithdrawalSuccessSum menjumlahkan tr_withdrawal.amount berstatus
	// SUCCESS dalam periode (berdasarkan completedDate) — outgoing §15.1,
	// sengaja dari tabel withdrawal langsung bukan ledger (lihat §15.1).
	WithdrawalSuccessSum(ctx context.Context, campaignID int64, from, to time.Time) (float64, error)
	// FeeSum menjumlahkan adminFee donasi PAID + fee withdrawal SUCCESS
	// dalam periode — kolom Fee §15.1.
	FeeSum(ctx context.Context, campaignID int64, from, to time.Time) (float64, error)

	// CampaignReportRows mengembalikan baris laporan campaign §15.2.
	CampaignReportRows(ctx context.Context, f report_dto.KantongAmalReportFilter) ([]report_dto.CampaignReportRow, int64, error)
	// DonationReportRows mengembalikan baris laporan donasi §15.3.
	DonationReportRows(ctx context.Context, f report_dto.KantongAmalReportFilter) ([]report_dto.DonationReportRow, int64, error)
	// WithdrawalReportRows mengembalikan baris laporan withdrawal §15.4.
	WithdrawalReportRows(ctx context.Context, f report_dto.KantongAmalReportFilter) ([]report_dto.WithdrawalReportRow, int64, error)
	// WithdrawalStatusFunnel mengembalikan breakdown jumlah per status §15.4.
	WithdrawalStatusFunnel(ctx context.Context, campaignID int64) ([]report_dto.WithdrawalStatusFunnel, error)

	// DonationPaidTotals mengembalikan jumlah & total nominal seluruh donasi
	// PAID (sepanjang waktu, bukan per-periode) — dipakai RunReconciliation.
	DonationPaidTotals(ctx context.Context) (count int64, amount float64, err error)
	// WithdrawalSuccessTotals sama seperti DonationPaidTotals untuk withdrawal SUCCESS.
	WithdrawalSuccessTotals(ctx context.Context) (count int64, amount float64, err error)
	// LedgerTotalByType menjumlahkan tr_wallet_ledger.amount untuk satu
	// entryType sepanjang waktu (seluruh campaign) — dipakai RunReconciliation.
	LedgerTotalByType(ctx context.Context, entryType string) (float64, error)

	// CreateReconciliationSnapshot menyimpan satu hasil jalan
	// finance.daily_reconciliation §15.5.
	CreateReconciliationSnapshot(ctx context.Context, p report_dto.ReconciliationSnapshotParams) (int64, error)
	// ListReconciliationSnapshots mengembalikan histori snapshot terbaru
	// lebih dulu, untuk melihat tren discrepancy dari waktu ke waktu.
	ListReconciliationSnapshots(ctx context.Context, q dto.ListQuery) ([]report_dto.ReconciliationSnapshotResponse, int64, error)

	// ListFinanceAuditLog mengembalikan histori tr_finance_audit_log §16.1,
	// terbaru lebih dulu.
	ListFinanceAuditLog(ctx context.Context, f report_dto.FinanceAuditLogFilter) ([]report_dto.FinanceAuditLogItem, int64, error)

	// GlobalLedgerRows mengembalikan baris debit/kredit global (item 6
	// revision-prompt-2.md) — langsung dari tr_wallet_ledger (khusus
	// Bisatopup by construction, lihat komentar report_dto.GlobalLedgerFilter).
	GlobalLedgerRows(ctx context.Context, f report_dto.GlobalLedgerFilter) ([]report_dto.GlobalLedgerRow, int64, error)

	// DonationAmountBands/DonorAgeBands mengembalikan distribusi donasi PAID
	// untuk tab Analitik (item 7 revision-prompt-2.md) — campaignID 0 berarti
	// seluruh campaign.
	DonationAmountBands(ctx context.Context, campaignID int64) ([]report_dto.AmountBandRow, error)
	DonorAgeBands(ctx context.Context, campaignID int64) ([]report_dto.AgeBandRow, error)
}
