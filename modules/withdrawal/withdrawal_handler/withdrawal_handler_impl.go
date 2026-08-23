package withdrawal_handler

import (
	"strconv"

	"fsldk-api/base/appctx"
	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/modules/withdrawal/withdrawal_dto"
	"fsldk-api/modules/withdrawal/withdrawal_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct{ svc withdrawal_service.Service }

// NewHandler membuat Handler withdrawal.
func NewHandler(svc withdrawal_service.Service) Handler { return &HandlerImpl{svc: svc} }

func idParam(c *gin.Context) (int64, error) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, apperror.BadRequest("ID tidak valid")
	}
	return id, nil
}

func (h *HandlerImpl) Request(c *gin.Context) {
	campaignID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || campaignID <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID campaign tidak valid"))
		return
	}
	var req withdrawal_dto.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	data, err := h.svc.Request(c.Request.Context(), campaignID, appctx.UserID(c), req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Created(c, "Permintaan penarikan berhasil dibuat", data)
}

func (h *HandlerImpl) MyList(c *gin.Context) {
	q := dto.ParseListQuery(c)
	data, total, err := h.svc.MyList(c.Request.Context(), appctx.UserID(c), q)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", httphelper.BuildPagination(c, data, total, q.Page, q.Limit))
}

func (h *HandlerImpl) Cancel(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	if err := h.svc.Cancel(c.Request.Context(), id, appctx.UserID(c)); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Permintaan penarikan dibatalkan", nil)
}

func (h *HandlerImpl) CMSList(c *gin.Context) {
	q := dto.ParseListQuery(c)
	data, total, err := h.svc.CMSList(c.Request.Context(), q, c.Query("status"))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", httphelper.BuildPagination(c, data, total, q.Page, q.Limit))
}

func (h *HandlerImpl) Approve(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	data, err := h.svc.Approve(c.Request.Context(), id, appctx.UserID(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Penarikan disetujui", data)
}

func (h *HandlerImpl) Reject(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	var req withdrawal_dto.RejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	if err := h.svc.Reject(c.Request.Context(), id, appctx.UserID(c), req.Reason); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Penarikan ditolak", nil)
}

func (h *HandlerImpl) Process(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	data, err := h.svc.Process(c.Request.Context(), id, appctx.UserID(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Pencairan diproses", data)
}

// Callback menerima webhook disbursement dari Bisabiller. Diamankan via
// URL path secret (bukan JWT, bukan signature — Bisabiller tidak mengirim
// field signature untuk callback transfer, lihat withdrawal_service.ProcessCallback).
func (h *HandlerImpl) Callback(c *gin.Context) {
	var req withdrawal_dto.DisbursementCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format callback tidak valid"))
		return
	}
	if err := h.svc.ProcessCallback(c.Request.Context(), req, c.Param("secret")); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "OK", nil)
}

func (h *HandlerImpl) Inquiry(c *gin.Context) {
	var req withdrawal_dto.InquiryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	data, err := h.svc.Inquiry(c.Request.Context(), req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) ListBanks(c *gin.Context) {
	data, err := h.svc.ListBanks(c.Request.Context())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}
