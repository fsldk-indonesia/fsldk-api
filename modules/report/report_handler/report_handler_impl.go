package report_handler

import (
	"fmt"
	"strconv"
	"time"

	"fsldk-api/base/appctx"
	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/httphelper"
	"fsldk-api/modules/report/report_dto"
	"fsldk-api/modules/report/report_service"

	"github.com/gin-gonic/gin"
)

// requestedOrganizationID membaca query `organizationID` opsional (target
// org-switcher) — divalidasi di service, bukan di sini.
func requestedOrganizationID(c *gin.Context) *int64 {
	raw := c.Query("organizationID")
	if raw == "" {
		return nil
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return nil
	}
	return &id
}

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct{ svc report_service.Service }

// NewHandler membuat Handler report.
func NewHandler(svc report_service.Service) Handler { return &HandlerImpl{svc: svc} }

func (h *HandlerImpl) ExportSubmissions(c *gin.Context) {
	formCode := c.Query("formCode")
	if formCode == "" {
		httphelper.Error(c, apperror.BadRequest("Parameter formCode wajib diisi"))
		return
	}
	format := c.DefaultQuery("format", "xlsx")

	caller := report_service.CallerScope{
		UserID:                  appctx.UserID(c),
		OrganizationID:          appctx.OrganizationID(c),
		OrganizationTypeCode:    appctx.OrganizationTypeCode(c),
		WildcardTierAccess:      appctx.WildcardTierAccess(c),
		RequestedOrganizationID: requestedOrganizationID(c),
	}
	result, err := h.svc.ExportSubmissions(c.Request.Context(), caller, report_dto.ExportFilter{
		FormCode: formCode,
		Status:   c.Query("status"),
		Format:   format,
	})
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", result.FileName))
	c.Data(200, result.ContentType, result.Data)
}

// ---------- Kantong Amal (Phase 9) ----------

func queryInt64(c *gin.Context, name string) int64 {
	v, _ := strconv.ParseInt(c.Query(name), 10, 64)
	return v
}

func queryDate(c *gin.Context, name string) (time.Time, bool) {
	raw := c.Query(name)
	if raw == "" {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func kantongAmalFilter(c *gin.Context) report_dto.KantongAmalReportFilter {
	q := dto.ParseListQuery(c)
	from, _ := queryDate(c, "from")
	to, hasTo := queryDate(c, "to")
	if hasTo {
		// Inklusif sampai akhir hari "to" (bukan tengah malamnya).
		to = to.Add(24*time.Hour - time.Second)
	}
	return report_dto.KantongAmalReportFilter{
		From: from, To: to, CampaignID: queryInt64(c, "campaignID"), Status: c.Query("status"),
		Page: q.Page, Limit: q.Limit,
	}
}

func (h *HandlerImpl) BalanceReport(c *gin.Context) {
	from, ok1 := queryDate(c, "from")
	to, ok2 := queryDate(c, "to")
	if !ok1 || !ok2 {
		httphelper.Error(c, apperror.BadRequest("Parameter from dan to (format YYYY-MM-DD) wajib diisi"))
		return
	}
	to = to.Add(24*time.Hour - time.Second)
	result, err := h.svc.GetBalanceReport(c.Request.Context(), report_dto.BalanceReportFilter{From: from, To: to, CampaignID: queryInt64(c, "campaignID")})
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", result)
}

func (h *HandlerImpl) ExportBalanceReport(c *gin.Context) {
	from, ok1 := queryDate(c, "from")
	to, ok2 := queryDate(c, "to")
	if !ok1 || !ok2 {
		httphelper.Error(c, apperror.BadRequest("Parameter from dan to (format YYYY-MM-DD) wajib diisi"))
		return
	}
	to = to.Add(24*time.Hour - time.Second)
	result, err := h.svc.ExportBalanceReport(c.Request.Context(), appctx.UserID(c), report_dto.BalanceReportFilter{From: from, To: to, CampaignID: queryInt64(c, "campaignID")})
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", result.FileName))
	c.Data(200, result.ContentType, result.Data)
}

func (h *HandlerImpl) CampaignReport(c *gin.Context) {
	data, total, err := h.svc.ListCampaignReport(c.Request.Context(), kantongAmalFilter(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	q := dto.ParseListQuery(c)
	httphelper.Success(c, "", httphelper.BuildPagination(c, data, total, q.Page, q.Limit))
}

func (h *HandlerImpl) ExportCampaignReport(c *gin.Context) {
	result, err := h.svc.ExportCampaignReport(c.Request.Context(), appctx.UserID(c), kantongAmalFilter(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", result.FileName))
	c.Data(200, result.ContentType, result.Data)
}

func (h *HandlerImpl) DonationReport(c *gin.Context) {
	data, total, err := h.svc.ListDonationReport(c.Request.Context(), kantongAmalFilter(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	q := dto.ParseListQuery(c)
	httphelper.Success(c, "", httphelper.BuildPagination(c, data, total, q.Page, q.Limit))
}

func (h *HandlerImpl) ExportDonationReport(c *gin.Context) {
	result, err := h.svc.ExportDonationReport(c.Request.Context(), appctx.UserID(c), kantongAmalFilter(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", result.FileName))
	c.Data(200, result.ContentType, result.Data)
}

func (h *HandlerImpl) WithdrawalReport(c *gin.Context) {
	data, total, funnel, err := h.svc.ListWithdrawalReport(c.Request.Context(), kantongAmalFilter(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	q := dto.ParseListQuery(c)
	page := httphelper.BuildPagination(c, data, total, q.Page, q.Limit)
	httphelper.Success(c, "", gin.H{"items": page, "statusFunnel": funnel})
}

func (h *HandlerImpl) ExportWithdrawalReport(c *gin.Context) {
	result, err := h.svc.ExportWithdrawalReport(c.Request.Context(), appctx.UserID(c), kantongAmalFilter(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", result.FileName))
	c.Data(200, result.ContentType, result.Data)
}

func (h *HandlerImpl) ReconciliationHistory(c *gin.Context) {
	q := dto.ParseListQuery(c)
	data, total, err := h.svc.ListReconciliationHistory(c.Request.Context(), q)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", httphelper.BuildPagination(c, data, total, q.Page, q.Limit))
}

// RunReconciliation memicu satu kali jalan finance.daily_reconciliation
// secara manual (di luar jadwal ticker) — berguna bagi admin yang ingin
// snapshot terbaru tanpa menunggu siklus harian berikutnya.
func (h *HandlerImpl) RunReconciliation(c *gin.Context) {
	result, err := h.svc.RunReconciliation(c.Request.Context())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Rekonsiliasi dijalankan", result)
}

func (h *HandlerImpl) FinanceAuditLog(c *gin.Context) {
	q := dto.ParseListQuery(c)
	data, total, err := h.svc.ListFinanceAuditLog(c.Request.Context(), report_dto.FinanceAuditLogFilter{
		Entity: c.Query("entity"), Action: c.Query("action"), Page: q.Page, Limit: q.Limit,
	})
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", httphelper.BuildPagination(c, data, total, q.Page, q.Limit))
}
