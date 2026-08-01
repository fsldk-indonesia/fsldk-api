package news_handler

import (
	"strconv"

	"fsldk-api/base/apperror"
	"fsldk-api/base/appctx"
	"fsldk-api/base/dto"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/constants"
	"fsldk-api/modules/news/news_dto"
	"fsldk-api/modules/news/news_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct{ svc news_service.Service }

// NewHandler membuat Handler berita.
func NewHandler(svc news_service.Service) Handler { return &HandlerImpl{svc: svc} }

func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return id, true
}

func (h *HandlerImpl) hasPublishPermission(c *gin.Context) bool {
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

func (h *HandlerImpl) PublicList(c *gin.Context) {
	q := dto.ParseListQuery(c)
	data, total, err := h.svc.PublicList(c.Request.Context(), q, c.Query("category"))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", httphelper.BuildPagination(c, data, total, q.Page, q.Limit))
}

func (h *HandlerImpl) PublicDetail(c *gin.Context) {
	data, err := h.svc.PublicDetail(c.Request.Context(), c.Param("slug"))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) PublicFeatured(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "5"))
	data, err := h.svc.Featured(c.Request.Context(), limit)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) Categories(c *gin.Context) {
	data, err := h.svc.Categories(c.Request.Context())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) CMSList(c *gin.Context) {
	q := dto.ParseListQuery(c)
	categoryID, _ := strconv.ParseInt(c.Query("categoryID"), 10, 64)
	data, total, err := h.svc.CMSList(c.Request.Context(), q, c.Query("status"), categoryID)
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
	data, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) bindRequest(c *gin.Context) (news_dto.Request, bool) {
	var req news_dto.Request
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

func (h *HandlerImpl) Create(c *gin.Context) {
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

func (h *HandlerImpl) Update(c *gin.Context) {
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

func (h *HandlerImpl) Publish(c *gin.Context) {
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

func (h *HandlerImpl) Featured(c *gin.Context) {
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

func (h *HandlerImpl) Delete(c *gin.Context) {
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
