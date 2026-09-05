// Package report_handler adalah lapisan presentasi HTTP modul report.
package report_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul report.
type Handler interface {
	ExportSubmissions(c *gin.Context)

	// ---------- Kantong Amal (Phase 9) ----------
	BalanceReport(c *gin.Context)
	ExportBalanceReport(c *gin.Context)
	CampaignReport(c *gin.Context)
	ExportCampaignReport(c *gin.Context)
	DonationReport(c *gin.Context)
	ExportDonationReport(c *gin.Context)
	WithdrawalReport(c *gin.Context)
	ExportWithdrawalReport(c *gin.Context)
	Reconciliation(c *gin.Context)
	FinanceAuditLog(c *gin.Context)
	GlobalLedger(c *gin.Context)
	Analytics(c *gin.Context)
}
