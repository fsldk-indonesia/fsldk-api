package role_handler

import (
	"strconv"

	"fsldk-api/base/apperror"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/modules/role/role_dto"
	"fsldk-api/modules/role/role_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct{ svc role_service.Service }

// NewHandler membuat Handler role.
func NewHandler(svc role_service.Service) Handler { return &HandlerImpl{svc: svc} }

func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return id, true
}

func (h *HandlerImpl) List(c *gin.Context) {
	data, err := h.svc.List(c.Request.Context(), c.Query("search"))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) Get(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	data, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) Create(c *gin.Context) {
	var req role_dto.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	data, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Created(c, "Role berhasil dibuat", data)
}

func (h *HandlerImpl) Update(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req role_dto.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	data, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Role berhasil diperbarui", data)
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
	httphelper.Success(c, "Role berhasil dihapus", nil)
}

func (h *HandlerImpl) SetPermissions(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req role_dto.SetPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	data, err := h.svc.SetPermissions(c.Request.Context(), id, req.PermissionIDs)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Permission role diperbarui", data)
}

func (h *HandlerImpl) Users(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	data, err := h.svc.Users(c.Request.Context(), id)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}
