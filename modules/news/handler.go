package news

import (
	"strconv"

	"fsldk-api/base/apperror"
	"fsldk-api/base/appctx"
	"fsldk-api/base/dto"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/constants"

	"github.com/gin-gonic/gin"
)

// Handler menangani request HTTP modul berita (publik & CMS).
type Handler struct{ svc *Service }

// NewHandler membuat Handler berita.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return id, true
}

func (h *Handler) hasPublishPermission(c *gin.Context) bool {
	if perms, ok := c.Get(constants.CtxPermissions); ok {
		if list, ok := perms.([]string); ok {
			for _, p := range list {
				if p == constants.PermNewsPublish {
					return true
				}
			}
		}
	}
	return false
}

// ----- Publik -----

// PublicList menangani GET /public/news.
func (h *Handler) PublicList(c *gin.Context) {
	q := dto.ParseListQuery(c)
	data, total, err := h.svc.PublicList(c.Request.Context(), q, c.Query("category"))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", httphelper.BuildPagination(c, data, total, q.Page, q.Limit))
}

// PublicDetail menangani GET /public/news/:slug.
func (h *Handler) PublicDetail(c *gin.Context) {
	data, err := h.svc.PublicDetail(c.Request.Context(), c.Param("slug"))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

// PublicFeatured menangani GET /public/news/featured.
func (h *Handler) PublicFeatured(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "5"))
	data, err := h.svc.Featured(c.Request.Context(), limit)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

// Categories menangani GET /public/news-categories.
func (h *Handler) Categories(c *gin.Context) {
	data, err := h.svc.Categories(c.Request.Context())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

// ----- CMS -----

// CMSList menangani GET /news.
func (h *Handler) CMSList(c *gin.Context) {
	q := dto.ParseListQuery(c)
	categoryID, _ := strconv.ParseInt(c.Query("categoryID"), 10, 64)
	data, total, err := h.svc.CMSList(c.Request.Context(), q, c.Query("status"), categoryID)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", httphelper.BuildPagination(c, data, total, q.Page, q.Limit))
}

// CMSGet menangani GET /news/:id.
func (h *Handler) CMSGet(c *gin.Context) {
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

func (h *Handler) bindRequest(c *gin.Context) (Request, bool) {
	var req Request
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

// Create menangani POST /news.
func (h *Handler) Create(c *gin.Context) {
	req, ok := h.bindRequest(c)
	if !ok {
		return
	}
	data, err := h.svc.Create(c.Request.Context(), req, appctx.UserID(c), h.hasPublishPermission(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Created(c, "Berita berhasil dibuat", data)
}

// Update menangani PUT /news/:id.
func (h *Handler) Update(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	req, ok := h.bindRequest(c)
	if !ok {
		return
	}
	data, err := h.svc.Update(c.Request.Context(), id, req, appctx.UserID(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Berita berhasil diperbarui", data)
}

// Publish menangani PATCH /news/:id/publish.
func (h *Handler) Publish(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var body struct {
		IsPublished bool `json:"isPublished"`
	}
	_ = c.ShouldBindJSON(&body)
	if err := h.svc.SetPublished(c.Request.Context(), id, body.IsPublished, appctx.UserID(c)); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Status publikasi diperbarui", nil)
}

// Featured menangani PATCH /news/:id/featured.
func (h *Handler) Featured(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var body struct {
		IsFeatured bool `json:"isFeatured"`
	}
	_ = c.ShouldBindJSON(&body)
	if err := h.svc.SetFeatured(c.Request.Context(), id, body.IsFeatured, appctx.UserID(c)); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Status unggulan diperbarui", nil)
}

// Delete menangani DELETE /news/:id.
func (h *Handler) Delete(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Berita berhasil dihapus", nil)
}
