package financeformat_handler

import (
	"strconv"
	"strings"

	"fsldk-api/base/appctx"
	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/modules/financeformat/financeformat_dto"
	"fsldk-api/modules/financeformat/financeformat_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl is the Handler implementation.
type HandlerImpl struct{ svc financeformat_service.Service }

// NewHandler creates the financeformat Handler.
func NewHandler(svc financeformat_service.Service) Handler { return &HandlerImpl{svc: svc} }

func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return id, true
}

func (h *HandlerImpl) bindRequest(c *gin.Context) (financeformat_dto.Request, bool) {
	var req financeformat_dto.Request
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
	data, err := h.svc.PublicList(c.Request.Context())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

// Download streams the Excel file with a Content-Disposition filename derived
// from the user-entered fileName (kebab-case + .xlsx), so both the download
// and a copied link resolve to a readable name instead of the random token.
func (h *HandlerImpl) Download(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	path, name, err := h.svc.PrepareDownload(c.Request.Context(), id)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	c.FileAttachment(path, name)
}

func (h *HandlerImpl) CMSList(c *gin.Context) {
	q := dto.ParseListQuery(c)
	f := financeformat_dto.Filter{
		FormatTypeID: parseInt64Query(c, "formatTypeID"),
		DateFrom:     strings.TrimSpace(c.Query("dateFrom")),
		DateTo:       strings.TrimSpace(c.Query("dateTo")),
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

func (h *HandlerImpl) FormatTypes(c *gin.Context) {
	data, err := h.svc.FormatTypes(c.Request.Context())
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
	httphelper.Created(c, "Format keuangan berhasil dibuat", data)
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
	httphelper.Success(c, "Format keuangan berhasil diperbarui", data)
}

func (h *HandlerImpl) Publish(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var body financeformat_dto.PublishRequest
	_ = c.ShouldBindJSON(&body)
	if err := h.svc.SetActive(c.Request.Context(), id, body.IsActive, appctx.UserID(c)); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Status format keuangan diperbarui", nil)
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
	httphelper.Success(c, "Format keuangan berhasil dihapus", nil)
}

// parseInt64Query reads a single optional int64 query param (0 when absent/invalid).
func parseInt64Query(c *gin.Context, key string) int64 {
	v, err := strconv.ParseInt(strings.TrimSpace(c.Query(key)), 10, 64)
	if err != nil {
		return 0
	}
	return v
}
