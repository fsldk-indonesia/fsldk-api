package article_handler

import (
	"strconv"
	"strings"

	"fsldk-api/base/appctx"
	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/constants"
	"fsldk-api/modules/article/article_dto"
	"fsldk-api/modules/article/article_service"

	"github.com/gin-gonic/gin"
)

// trimmedNonEmpty trims each value and drops blanks — multi-select filter
// params arrive as repeated query keys (?category=a&category=b, parsed via
// c.QueryArray), not comma-joined, mirroring the same convention used by the
// Galeri module's public filter (gallery_handler_impl.go).
func trimmedNonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// parseInts best-effort parses each value as an int, skipping invalid ones.
func parseInts(values []string) []int {
	out := make([]int, 0, len(values))
	for _, v := range values {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			out = append(out, n)
		}
	}
	return out
}

// parseBoolPtr parses "true"/"1" or "false"/"0" into a *bool — nil (meaning
// "no filter") for an absent or unrecognized value, distinguishing the
// "Punya PDF" filter's three states (semua / ya / tidak) without a sentinel.
func parseBoolPtr(v string) *bool {
	switch strings.TrimSpace(v) {
	case "true", "1":
		b := true
		return &b
	case "false", "0":
		b := false
		return &b
	default:
		return nil
	}
}

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct{ svc article_service.Service }

// NewHandler membuat Handler artikel.
func NewHandler(svc article_service.Service) Handler { return &HandlerImpl{svc: svc} }

func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return id, true
}

func (h *HandlerImpl) canPublish(c *gin.Context) bool {
	if perms, ok := c.Get(constants.CtxPermissions); ok {
		if list, ok := perms.([]string); ok {
			for _, p := range list {
				if p == constants.PermArticlePublish {
					return true
				}
			}
		}
	}
	return false
}

func (h *HandlerImpl) bindRequest(c *gin.Context) (article_dto.Request, bool) {
	var req article_dto.Request
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
	f := article_dto.PublicFilter{
		CategorySlugs: trimmedNonEmpty(c.QueryArray("category")),
		Years:         parseInts(c.QueryArray("year")),
		Writers:       trimmedNonEmpty(c.QueryArray("writer")),
		Months:        parseInts(c.QueryArray("month")),
		HasPdf:        parseBoolPtr(c.Query("hasPdf")),
	}
	data, total, err := h.svc.PublicList(c.Request.Context(), q, f)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", httphelper.BuildPagination(c, data, total, q.Page, q.Limit))
}

func (h *HandlerImpl) FilterOptionsPublic(c *gin.Context) {
	data, err := h.svc.FilterOptionsPublic(c.Request.Context())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) PublicDetail(c *gin.Context) {
	data, err := h.svc.PublicDetail(c.Request.Context(), c.Param("slug"))
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
	f := article_dto.CMSFilter{
		Status:      dto.ParseCSV(c.Query("status")),
		CategoryIDs: dto.ParseInt64CSV(c.Query("category")),
		Writer:      c.Query("writer"),
		DateFrom:    c.Query("dateFrom"),
		DateTo:      c.Query("dateTo"),
	}
	data, total, err := h.svc.CMSList(c.Request.Context(), q, f)
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
	data, err := h.svc.Create(c.Request.Context(), req, appctx.UserID(c), h.canPublish(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Created(c, "Artikel berhasil dibuat", data)
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
	httphelper.Success(c, "Artikel berhasil diperbarui", data)
}

func (h *HandlerImpl) Publish(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var body article_dto.PublishRequest
	_ = c.ShouldBindJSON(&body)
	if err := h.svc.SetPublished(c.Request.Context(), id, body.IsPublished, appctx.UserID(c)); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Status publikasi diperbarui", nil)
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
	httphelper.Success(c, "Artikel berhasil dihapus", nil)
}

func (h *HandlerImpl) BulkDelete(c *gin.Context) {
	var req article_dto.BulkDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	if err := h.svc.BulkDelete(c.Request.Context(), req.IDs); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Artikel terpilih berhasil dihapus", nil)
}
