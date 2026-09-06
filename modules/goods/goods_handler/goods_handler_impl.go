package goods_handler

import (
	"strconv"
	"strings"

	"fsldk-api/base/appctx"
	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/modules/goods/goods_dto"
	"fsldk-api/modules/goods/goods_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct{ svc goods_service.Service }

// NewHandler membuat Handler goods.
func NewHandler(svc goods_service.Service) Handler { return &HandlerImpl{svc: svc} }

func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return id, true
}

func (h *HandlerImpl) parseFilter(c *gin.Context) goods_dto.Filter {
	categoryID, _ := strconv.ParseInt(c.Query("categoryID"), 10, 64)
	return goods_dto.Filter{
		CategorySlug: strings.TrimSpace(c.Query("category")),
		CategoryID:   categoryID,
		Availability: strings.TrimSpace(c.Query("availability")),
		FeaturedOnly: c.Query("featured") == "1" || c.Query("featured") == "true",
	}
}

func (h *HandlerImpl) bindRequest(c *gin.Context) (goods_dto.Request, bool) {
	var req goods_dto.Request
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

func (h *HandlerImpl) bindCategoryRequest(c *gin.Context) (goods_dto.CategoryRequest, bool) {
	var req goods_dto.CategoryRequest
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

func (h *HandlerImpl) PublicList(c *gin.Context) {
	q := dto.ParseListQuery(c)
	sort := c.DefaultQuery("sort", "newest")
	data, total, err := h.svc.PublicList(c.Request.Context(), q, h.parseFilter(c), sort)
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

func (h *HandlerImpl) PublicCategories(c *gin.Context) {
	data, err := h.svc.PublicCategories(c.Request.Context())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) CMSList(c *gin.Context) {
	q := dto.ParseListQuery(c)
	data, total, err := h.svc.CMSList(c.Request.Context(), q, h.parseFilter(c))
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

func (h *HandlerImpl) Create(c *gin.Context) {
	req, ok := h.bindRequest(c)
	if !ok {
		return
	}
	data, err := h.svc.Create(c.Request.Context(), req, appctx.UserID(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Created(c, "Produk berhasil dibuat", data)
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
	httphelper.Success(c, "Produk berhasil diperbarui", data)
}

func (h *HandlerImpl) Publish(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var body goods_dto.PublishRequest
	_ = c.ShouldBindJSON(&body)
	if err := h.svc.SetPublished(c.Request.Context(), id, body.IsPublished, appctx.UserID(c)); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Status publikasi diperbarui", nil)
}

func (h *HandlerImpl) SetFeatured(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var body goods_dto.FeaturedRequest
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
	httphelper.Success(c, "Produk berhasil dihapus", nil)
}

func (h *HandlerImpl) CategoryList(c *gin.Context) {
	data, err := h.svc.CMSCategories(c.Request.Context())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) CategoryCreate(c *gin.Context) {
	req, ok := h.bindCategoryRequest(c)
	if !ok {
		return
	}
	data, err := h.svc.CategoryCreate(c.Request.Context(), req, appctx.UserID(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Created(c, "Kategori berhasil dibuat", data)
}

func (h *HandlerImpl) CategoryUpdate(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	req, ok := h.bindCategoryRequest(c)
	if !ok {
		return
	}
	data, err := h.svc.CategoryUpdate(c.Request.Context(), id, req, appctx.UserID(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Kategori berhasil diperbarui", data)
}

func (h *HandlerImpl) CategoryDelete(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.CategoryDelete(c.Request.Context(), id); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Kategori berhasil dihapus", nil)
}
