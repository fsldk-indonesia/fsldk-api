package event_handler

import (
	"strconv"
	"strings"

	"fsldk-api/base/appctx"
	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/modules/event/event_dto"
	"fsldk-api/modules/event/event_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl is the concrete implementation of Handler.
type HandlerImpl struct{ svc event_service.Service }

// NewHandler creates a Handler backed by the given Service.
func NewHandler(svc event_service.Service) Handler { return &HandlerImpl{svc: svc} }

// idParam parses and validates the :id route parameter.
func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return id, true
}

// splitQuery splits a comma-separated query param into a non-empty string slice.
func splitQuery(v string) []string {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// bindRequest decodes and validates the JSON body into CreateRequest.
func bindRequest(c *gin.Context) (event_dto.CreateRequest, bool) {
	var req event_dto.CreateRequest
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

// --- Public handlers ---

func (h *HandlerImpl) ListPublic(c *gin.Context) {
	q := dto.ParseListQuery(c)
	q.Limit = 9 // default page size for public event grid
	if l, _ := strconv.Atoi(c.Query("limit")); l > 0 && l <= 100 {
		q.Limit = l
	}
	divisions := splitQuery(c.Query("division"))
	years := splitQuery(c.Query("year"))
	statuses := splitQuery(c.Query("status"))
	sort := c.DefaultQuery("sort", "newest")

	data, total, err := h.svc.PublicList(c.Request.Context(), q, divisions, years, statuses, sort)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", httphelper.BuildPagination(c, data, total, q.Page, q.Limit))
}

func (h *HandlerImpl) ShowPublic(c *gin.Context) {
	data, err := h.svc.PublicDetail(c.Request.Context(), c.Param("slug"))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

// --- CMS handlers ---

func (h *HandlerImpl) ListCMS(c *gin.Context) {
	q := dto.ParseListQuery(c)
	data, total, err := h.svc.CMSList(c.Request.Context(), q, c.Query("division"))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", httphelper.BuildPagination(c, data, total, q.Page, q.Limit))
}

func (h *HandlerImpl) ShowCMS(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	data, err := h.svc.CMSGet(c.Request.Context(), id)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) Create(c *gin.Context) {
	req, ok := bindRequest(c)
	if !ok {
		return
	}
	data, err := h.svc.Create(c.Request.Context(), req, appctx.UserID(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Created(c, "Event berhasil dibuat", data)
}

func (h *HandlerImpl) Update(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	req, ok := bindRequest(c)
	if !ok {
		return
	}
	data, err := h.svc.Update(c.Request.Context(), id, req, appctx.UserID(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Event berhasil diperbarui", data)
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
	httphelper.Success(c, "Event berhasil dihapus", nil)
}
