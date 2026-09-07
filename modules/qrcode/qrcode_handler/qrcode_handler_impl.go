package qrcode_handler

import (
	"net/http"
	"strconv"

	"fsldk-api/base/appctx"
	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/modules/qrcode/qrcode_dto"
	"fsldk-api/modules/qrcode/qrcode_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct{ svc qrcode_service.Service }

// NewHandler membuat Handler QR Code.
func NewHandler(svc qrcode_service.Service) Handler { return &HandlerImpl{svc: svc} }

// batas ukuran gambar QR (piksel) yang boleh diminta lewat query ?size=.
const (
	minImageSize     = 128
	maxImageSize     = 1024
	defaultImageSize = 512
)

func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return id, true
}

func (h *HandlerImpl) List(c *gin.Context) {
	q := dto.ParseListQuery(c)
	data, total, err := h.svc.List(c.Request.Context(), q)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", httphelper.BuildPagination(c, data, total, q.Page, q.Limit))
}

func (h *HandlerImpl) Get(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	res, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", res)
}

func (h *HandlerImpl) Create(c *gin.Context) {
	var req qrcode_dto.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	res, err := h.svc.Create(c.Request.Context(), req, appctx.UserID(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Created(c, "QR Code berhasil dibuat", res)
}

func (h *HandlerImpl) Update(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req qrcode_dto.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	res, err := h.svc.Update(c.Request.Context(), id, req, appctx.UserID(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "QR Code berhasil diperbarui", res)
}

func (h *HandlerImpl) Delete(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "QR Code berhasil dihapus", nil)
}

func (h *HandlerImpl) Image(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	size := defaultImageSize
	if raw := c.Query("size"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			size = n
		}
	}
	if size < minImageSize {
		size = minImageSize
	}
	if size > maxImageSize {
		size = maxImageSize
	}

	png, err := h.svc.Image(c.Request.Context(), id, size)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	c.Data(http.StatusOK, "image/png", png)
}
