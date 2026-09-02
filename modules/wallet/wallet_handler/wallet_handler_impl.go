package wallet_handler

import (
	"strconv"
	"time"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/httphelper"
	"fsldk-api/modules/wallet/wallet_dto"
	"fsldk-api/modules/wallet/wallet_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct{ svc wallet_service.Service }

// NewHandler membuat Handler wallet.
func NewHandler(svc wallet_service.Service) Handler { return &HandlerImpl{svc: svc} }

func campaignIDParam(c *gin.Context) (int64, error) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, apperror.BadRequest("ID campaign tidak valid")
	}
	return id, nil
}

// ledgerFilter membaca filter riwayat ledger dari query string. dateFrom/
// dateTo memakai format tanggal sederhana (YYYY-MM-DD) — bila gagal
// diparsing, filter tanggal tersebut diabaikan (bukan error), konsisten
// dengan dto.ParseListQuery yang juga selalu punya fallback aman.
func ledgerFilter(c *gin.Context) wallet_dto.LedgerListFilter {
	q := dto.ParseListQuery(c)
	f := wallet_dto.LedgerListFilter{EntryType: c.Query("entryType"), Page: q.Page, Limit: q.Limit}
	if v := c.Query("dateFrom"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			f.DateFrom = &t
		}
	}
	if v := c.Query("dateTo"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			t = t.Add(24*time.Hour - time.Second) // inklusif sampai akhir hari
			f.DateTo = &t
		}
	}
	return f
}

func (h *HandlerImpl) CMSBalance(c *gin.Context) {
	campaignID, err := campaignIDParam(c)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	data, err := h.svc.GetBalance(c.Request.Context(), campaignID)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) CMSLedger(c *gin.Context) {
	campaignID, err := campaignIDParam(c)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	f := ledgerFilter(c)
	data, total, err := h.svc.ListLedger(c.Request.Context(), campaignID, f)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", httphelper.BuildPagination(c, data, int(total), f.Page, f.Limit))
}
