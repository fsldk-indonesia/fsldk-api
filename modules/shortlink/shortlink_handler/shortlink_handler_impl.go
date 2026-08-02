package shortlink_handler

import (
	"strconv"

	"fsldk-api/base/appctx"
	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/modules/shortlink/shortlink_dto"
	"fsldk-api/modules/shortlink/shortlink_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct{ svc shortlink_service.Service }

// NewHandler membuat Handler shortlink.
func NewHandler(svc shortlink_service.Service) Handler { return &HandlerImpl{svc: svc} }

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
	var req shortlink_dto.CreateRequest
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
	httphelper.Created(c, "Shortlink berhasil dibuat", res)
}

func (h *HandlerImpl) Update(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req shortlink_dto.UpdateRequest
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
	httphelper.Success(c, "Shortlink berhasil diperbarui", res)
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
	httphelper.Success(c, "Shortlink berhasil dihapus", nil)
}

func (h *HandlerImpl) Resolve(c *gin.Context) {
	destination, err := h.svc.Resolve(c.Request.Context(), c.Param("key"))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", shortlink_dto.ResolveResponse{DestinationURL: destination})
}
