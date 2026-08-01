package content

import (
	"strconv"

	"fsldk-api/base/apperror"
	"fsldk-api/base/appctx"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"

	"github.com/gin-gonic/gin"
)

// Handler menangani request HTTP modul content.
type Handler struct{ svc *Service }

// NewHandler membuat Handler content.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return id, true
}

// ----- Publik -----

// PublicList menangani GET /public/contents.
func (h *Handler) PublicList(c *gin.Context) {
	data, err := h.svc.ListContent(c.Request.Context(), true)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

// PublicByKey menangani GET /public/contents/:key.
func (h *Handler) PublicByKey(c *gin.Context) {
	data, err := h.svc.GetContent(c.Request.Context(), c.Param("key"))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

// PublicProfile menangani GET /public/profile.
func (h *Handler) PublicProfile(c *gin.Context) {
	data, err := h.svc.Profile(c.Request.Context())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

// PublicOrg menangani GET /public/organization-structure.
func (h *Handler) PublicOrg(c *gin.Context) {
	data, err := h.svc.ListOrg(c.Request.Context(), true)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

// ----- CMS -----

// CMSList menangani GET /contents.
func (h *Handler) CMSList(c *gin.Context) {
	data, err := h.svc.ListContent(c.Request.Context(), false)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

// Update menangani PUT /contents/:key.
func (h *Handler) Update(c *gin.Context) {
	var req ContentUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	if err := h.svc.UpdateContent(c.Request.Context(), c.Param("key"), req, appctx.UserID(c)); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Konten berhasil diperbarui", nil)
}

// OrgList menangani GET /organization-structure.
func (h *Handler) OrgList(c *gin.Context) {
	data, err := h.svc.ListOrg(c.Request.Context(), false)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *Handler) bindOrg(c *gin.Context) (OrgRequest, bool) {
	var req OrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return req, false
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return req, false
	}
	return req, true
}

// OrgCreate menangani POST /organization-structure.
func (h *Handler) OrgCreate(c *gin.Context) {
	req, ok := h.bindOrg(c)
	if !ok {
		return
	}
	id, err := h.svc.CreateOrg(c.Request.Context(), req, appctx.UserID(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Created(c, "Pengurus berhasil ditambahkan", gin.H{"structureID": id})
}

// OrgUpdate menangani PUT /organization-structure/:id.
func (h *Handler) OrgUpdate(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	req, ok := h.bindOrg(c)
	if !ok {
		return
	}
	if err := h.svc.UpdateOrg(c.Request.Context(), id, req); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Pengurus berhasil diperbarui", nil)
}

// OrgDelete menangani DELETE /organization-structure/:id.
func (h *Handler) OrgDelete(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteOrg(c.Request.Context(), id); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Pengurus berhasil dihapus", nil)
}
