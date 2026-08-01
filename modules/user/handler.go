package user

import (
	"strconv"

	"fsldk-api/base/apperror"
	"fsldk-api/base/appctx"
	"fsldk-api/base/dto"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"

	"github.com/gin-gonic/gin"
)

// Handler menangani request HTTP modul user.
type Handler struct{ svc *Service }

// NewHandler membuat Handler user.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return id, true
}

// List menangani GET /users.
func (h *Handler) List(c *gin.Context) {
	q := dto.ParseListQuery(c)
	roleID, _ := strconv.ParseInt(c.Query("roleID"), 10, 64)
	data, total, err := h.svc.List(c.Request.Context(), q, roleID)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", httphelper.BuildPagination(c, data, total, q.Page, q.Limit))
}

// Get menangani GET /users/:id.
func (h *Handler) Get(c *gin.Context) {
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

// Create menangani POST /users.
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
	res, err := h.svc.Create(c.Request.Context(), req, appctx.UserID(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Created(c, "Pengguna berhasil dibuat", res)
}

// Update menangani PUT /users/:id.
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
	res, err := h.svc.Update(c.Request.Context(), id, req, appctx.UserID(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Pengguna berhasil diperbarui", res)
}

// SetStatus menangani PATCH /users/:id/status.
func (h *Handler) SetStatus(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req StatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := h.svc.SetStatus(c.Request.Context(), id, req.IsActive, appctx.UserID(c)); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Status pengguna diperbarui", nil)
}

// ResetPassword menangani POST /users/:id/reset-password.
func (h *Handler) ResetPassword(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	temp, err := h.svc.ResetPassword(c.Request.Context(), id)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Password berhasil direset", gin.H{"temporaryPassword": temp})
}

// Delete menangani DELETE /users/:id.
func (h *Handler) Delete(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id, appctx.UserID(c)); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Pengguna berhasil dihapus", nil)
}
