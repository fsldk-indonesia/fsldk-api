package qrcoderequest_handler

import (
	"strconv"

	"fsldk-api/base/appctx"
	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/modules/qrcode/qrcoderequest_dto"
	"fsldk-api/modules/qrcode/qrcoderequest_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct{ svc qrcoderequest_service.Service }

// NewHandler membuat Handler QR Code request.
func NewHandler(svc qrcoderequest_service.Service) Handler { return &HandlerImpl{svc: svc} }

func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return id, true
}

func (h *HandlerImpl) Submit(c *gin.Context) {
	var req qrcoderequest_dto.SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	res, err := h.svc.Submit(c.Request.Context(), req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Created(c, "Permintaan terkirim, kami akan mengabari lewat WhatsApp & email begitu diproses", res)
}

func (h *HandlerImpl) PublicPIC(c *gin.Context) {
	res, err := h.svc.PublicPIC(c.Request.Context())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", res)
}

func (h *HandlerImpl) CMSList(c *gin.Context) {
	q := dto.ParseListQuery(c)
	status := c.Query("status")
	data, total, err := h.svc.CMSList(c.Request.Context(), q, status)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", httphelper.BuildPagination(c, data, total, q.Page, q.Limit))
}

func (h *HandlerImpl) CMSGet(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	res, err := h.svc.CMSGet(c.Request.Context(), id)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", res)
}

func (h *HandlerImpl) Approve(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	res, err := h.svc.Approve(c.Request.Context(), id, appctx.UserID(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Permintaan disetujui, QR Code berhasil dibuat", res)
}

func (h *HandlerImpl) Reject(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req qrcoderequest_dto.RejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	if err := h.svc.Reject(c.Request.Context(), id, appctx.UserID(c), req.RejectionReason); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Permintaan ditolak", nil)
}
