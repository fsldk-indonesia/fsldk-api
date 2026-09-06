// Package report merangkai routing modul report.
package report

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/report/report_handler"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes mendaftarkan endpoint modul report. Kantong Amal selalu
// aktif sejak revisi 2026-09-01 (item 9 revision-prompt-2.md —
// KANTONG_AMAL_ENABLED dihapus, tidak ada lagi gating flag di sini).
// Endpoint /submissions/export (report existing, tidak terkait Kantong
// Amal) tidak terpengaruh.
func RegisterRoutes(rg *gin.RouterGroup, h report_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/reports")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("/submissions/export", mw.RequirePermission(
			constants.PermReportRegionExport, constants.PermReportNationalExport,
		), h.ExportSubmissions)

		// Kantong Amal (Phase 9, §15 techspec) — akses dibatasi murni via
		// permission (least-privilege), bukan org-scope seperti report submission.
		ka := g.Group("/kantong-amal")
		{
			ka.GET("/balance", mw.RequirePermission(constants.PermFinanceReportView), h.BalanceReport)
			ka.GET("/balance/export", mw.RequirePermission(constants.PermFinanceReportExport), h.ExportBalanceReport)
			ka.GET("/campaigns", mw.RequirePermission(constants.PermFinanceReportView), h.CampaignReport)
			ka.GET("/campaigns/export", mw.RequirePermission(constants.PermFinanceReportExport), h.ExportCampaignReport)
			ka.GET("/donations", mw.RequirePermission(constants.PermFinanceReportView), h.DonationReport)
			ka.GET("/donations/export", mw.RequirePermission(constants.PermFinanceReportExport), h.ExportDonationReport)
			ka.GET("/withdrawals", mw.RequirePermission(constants.PermFinanceReportView), h.WithdrawalReport)
			ka.GET("/withdrawals/export", mw.RequirePermission(constants.PermFinanceReportExport), h.ExportWithdrawalReport)
			ka.GET("/reconciliation", mw.RequirePermission(constants.PermFinanceReportView), h.Reconciliation)
			ka.GET("/audit-log", mw.RequirePermission(constants.PermFinanceAuditView), h.FinanceAuditLog)
			// Item 6/7 revision-prompt-2.md — debit/kredit global & Analitik.
			ka.GET("/ledger-global", mw.RequirePermission(constants.PermFinanceReportView), h.GlobalLedger)
			ka.GET("/analytics", mw.RequirePermission(constants.PermFinanceReportView), h.Analytics)
		}
	}
}
