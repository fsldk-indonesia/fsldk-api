package catalogbook_handler

import (
	"strconv"
	"strings"

	"fsldk-api/base/appctx"
	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/modules/catalogbook/catalogbook_dto"
	"fsldk-api/modules/catalogbook/catalogbook_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl is the Handler implementation.
type HandlerImpl struct{ svc catalogbook_service.Service }

// NewHandler creates the catalogbook Handler.
func NewHandler(svc catalogbook_service.Service) Handler { return &HandlerImpl{svc: svc} }

func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return id, true
}

// parseInt64List reads a query param that may repeat (?key=1&key=2) or be
// comma-separated (?key=1,2) into []int64.
func parseInt64List(c *gin.Context, key string) []int64 {
	var out []int64
	for _, raw := range c.QueryArray(key) {
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if v, err := strconv.ParseInt(part, 10, 64); err == nil {
				out = append(out, v)
			}
		}
	}
	return out
}

// parseStringList reads a query param that may repeat or be comma-separated
// into []string.
func parseStringList(c *gin.Context, key string) []string {
	var out []string
	for _, raw := range c.QueryArray(key) {
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}

func (h *HandlerImpl) parseFilter(c *gin.Context) catalogbook_dto.Filter {
	return catalogbook_dto.Filter{
		BookCategoryIDs:     parseInt64List(c, "bookCategoryID"),
		AuthorTypeIDs:       parseInt64List(c, "authorTypeID"),
		AvailabilityTypeIDs: parseInt64List(c, "availabilityTypeID"),
		LanguageIDs:         parseInt64List(c, "languageID"),
		Years:               parseStringList(c, "year"),
		Author:              strings.TrimSpace(c.Query("author")),
		Publisher:           strings.TrimSpace(c.Query("publisher")),
	}
}

func (h *HandlerImpl) bindRequest(c *gin.Context) (catalogbook_dto.Request, bool) {
	var req catalogbook_dto.Request
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

func (h *HandlerImpl) Like(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	count, err := h.svc.Like(c.Request.Context(), id)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Buku disukai", catalogbook_dto.LikeResponse{FavoriteCount: count})
}

func (h *HandlerImpl) Categories(c *gin.Context) {
	data, err := h.svc.Categories(c.Request.Context())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) Languages(c *gin.Context) {
	data, err := h.svc.Languages(c.Request.Context())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) AuthorTypes(c *gin.Context) {
	data, err := h.svc.AuthorTypes(c.Request.Context())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) AvailabilityTypes(c *gin.Context) {
	data, err := h.svc.AvailabilityTypes(c.Request.Context())
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
	httphelper.Created(c, "Buku berhasil dibuat", data)
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
	httphelper.Success(c, "Buku berhasil diperbarui", data)
}

func (h *HandlerImpl) Publish(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var body catalogbook_dto.PublishRequest
	_ = c.ShouldBindJSON(&body)
	if err := h.svc.SetActive(c.Request.Context(), id, body.IsActive, appctx.UserID(c)); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Status buku diperbarui", nil)
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
	httphelper.Success(c, "Buku berhasil dihapus", nil)
}
