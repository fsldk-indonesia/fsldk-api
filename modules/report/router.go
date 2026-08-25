// Package report merangkai routing modul report.
package report

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/report/report_handler"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes mendaftarkan endpoint modul report.
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
			ka.GET("/reconciliation", mw.RequirePermission(constants.PermFinanceReportView), h.ReconciliationHistory)
			ka.POST("/reconciliation/run", mw.RequirePermission(constants.PermFinanceReportExport), h.RunReconciliation)
		}
	}
}
