// Package report_service memuat logika bisnis modul report: ekspor laporan
// submission (Excel/CSV) dengan cakupan data identik dashboard (Section 28).
package report_service

import (
	"context"

	"fsldk-api/modules/report/report_dto"
)

// CallerScope menampung identitas & scope organisasi pengguna pemanggil.
type CallerScope struct {
	UserID               int64
	OrganizationID       *int64
	OrganizationTypeCode string
	WildcardTierAccess   string
	// RequestedOrganizationID adalah organizationID eksplisit dari query
	// `organizationID` (org-switcher shell cms-ldk/cms-puskomda) — divalidasi
	// accessible via OrgScopeResolver.IsAccessible sebelum dipakai.
	RequestedOrganizationID *int64
}

// OrgScopeResolver menyediakan resolusi cakupan akses organisasi berjenjang.
type OrgScopeResolver interface {
	IsAccessible(ctx context.Context, callerOrganizationID *int64, callerOrganizationTypeCode, wildcardTierAccess string, targetOrganizationID int64) (bool, error)
	AccessibleOrganizationIDs(ctx context.Context, callerOrganizationID *int64, callerOrganizationTypeCode, wildcardTierAccess string) ([]int64, error)
	AccessibleOrganizationIDsForTarget(ctx context.Context, targetOrganizationID int64) ([]int64, error)
}

// Service adalah kontrak logika bisnis report.
type Service interface {
	ExportSubmissions(ctx context.Context, caller CallerScope, filter report_dto.ExportFilter) (report_dto.ExportResult, error)

	// ---------- Kantong Amal (Phase 9) — §15 techspec ----------

	// GetBalanceReport menghitung ringkasan saldo satu periode §15.1,
	// tervalidasi otomatis terhadap ledger (isBalanced).
	GetBalanceReport(ctx context.Context, f report_dto.BalanceReportFilter) (report_dto.BalanceReportResponse, error)
	ExportBalanceReport(ctx context.Context, actorUserID int64, f report_dto.BalanceReportFilter) (report_dto.ExportResult, error)

	ListCampaignReport(ctx context.Context, f report_dto.KantongAmalReportFilter) ([]report_dto.CampaignReportRow, int, error)
	ExportCampaignReport(ctx context.Context, actorUserID int64, f report_dto.KantongAmalReportFilter) (report_dto.ExportResult, error)

	ListDonationReport(ctx context.Context, f report_dto.KantongAmalReportFilter) ([]report_dto.DonationReportRow, int, error)
	ExportDonationReport(ctx context.Context, actorUserID int64, f report_dto.KantongAmalReportFilter) (report_dto.ExportResult, error)

	// ListWithdrawalReport juga mengembalikan funnel breakdown per status §15.4.
	ListWithdrawalReport(ctx context.Context, f report_dto.KantongAmalReportFilter) ([]report_dto.WithdrawalReportRow, int, []report_dto.WithdrawalStatusFunnel, error)
	ExportWithdrawalReport(ctx context.Context, actorUserID int64, f report_dto.KantongAmalReportFilter) (report_dto.ExportResult, error)

	// GetReconciliation menghitung perbandingan lima sumber §15.5 LIVE saat
	// dipanggil (real-time, tidak disimpan — Balance Report tidak lagi
	// berbasis snapshot histori, lihat ReconciliationResponse).
	GetReconciliation(ctx context.Context) (report_dto.ReconciliationResponse, error)
	// ListFinanceAuditLog mengembalikan histori audit trail finance §16.1.
	ListFinanceAuditLog(ctx context.Context, f report_dto.FinanceAuditLogFilter) ([]report_dto.FinanceAuditLogItem, int, error)

	// ListGlobalLedger mengembalikan laporan debit/kredit global (item 6
	// revision-prompt-2.md).
	ListGlobalLedger(ctx context.Context, f report_dto.GlobalLedgerFilter) ([]report_dto.GlobalLedgerRow, int, error)
	// GetAnalytics mengembalikan data tab Analitik (item 7
	// revision-prompt-2.md) — distribusi nominal donasi, distribusi usia
	// donor, dan progres campaign, sekali panggil.
	GetAnalytics(ctx context.Context, campaignID int64) (report_dto.AnalyticsResponse, error)
}
