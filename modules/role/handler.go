package role

import (
	"strconv"

	"fsldk-api/base/apperror"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"

	"github.com/gin-gonic/gin"
)

// Handler menangani request HTTP modul role.
type Handler struct{ svc *Service }

// NewHandler membuat Handler role.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return id, true
}

// List menangani GET /roles.
func (h *Handler) List(c *gin.Context) {
	data, err := h.svc.List(c.Request.Context(), c.Query("search"))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

// Get menangani GET /roles/:id.
func (h *Handler) Get(c *gin.Context) {
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

// Create menangani POST /roles.
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
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

// Update menangani PUT /roles/:id.
func (h *Handler) Update(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req UpdateRequest
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

// Delete menangani DELETE /roles/:id.
func (h *Handler) Delete(c *gin.Context) {
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

// SetPermissions menangani PUT /roles/:id/permissions.
func (h *Handler) SetPermissions(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req SetPermissionsRequest
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

// Users menangani GET /roles/:id/users.
func (h *Handler) Users(c *gin.Context) {
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
