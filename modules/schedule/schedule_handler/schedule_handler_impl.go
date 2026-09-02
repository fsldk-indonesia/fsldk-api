package schedule_handler

import (
	"strconv"
	"strings"
	"time"

	"fsldk-api/base/appctx"
	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/modules/schedule/schedule_dto"
	"fsldk-api/modules/schedule/schedule_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl is the Handler implementation.
type HandlerImpl struct{ svc schedule_service.Service }

// NewHandler creates the schedule Handler.
func NewHandler(svc schedule_service.Service) Handler { return &HandlerImpl{svc: svc} }

func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return id, true
}

func (h *HandlerImpl) bindRequest(c *gin.Context) (schedule_dto.Request, bool) {
	var req schedule_dto.Request
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

// PublicList resolves the date window from ?from/?to, falling back to
// ?month/?year, then to the current month, and returns a plain array (no
// pagination) of active schedules overlapping it.
func (h *HandlerImpl) PublicList(c *gin.Context) {
	from := strings.TrimSpace(c.Query("from"))
	to := strings.TrimSpace(c.Query("to"))
	if from == "" || to == "" {
		now := time.Now()
		year, _ := strconv.Atoi(c.Query("year"))
		month, _ := strconv.Atoi(c.Query("month"))
		if year <= 0 {
			year = now.Year()
		}
		if month < 1 || month > 12 {
			month = int(now.Month())
		}
		first := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
		from = first.Format("2006-01-02")
		to = first.AddDate(0, 1, -1).Format("2006-01-02")
	}
	data, err := h.svc.PublicList(c.Request.Context(), from, to)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) parseFilter(c *gin.Context) schedule_dto.Filter {
	f := schedule_dto.Filter{
		Category: strings.TrimSpace(c.Query("category")),
		DateFrom: strings.TrimSpace(c.Query("dateFrom")),
		DateTo:   strings.TrimSpace(c.Query("dateTo")),
	}
	if v, err := strconv.Atoi(c.Query("month")); err == nil {
		f.Month = v
	}
	if v, err := strconv.Atoi(c.Query("year")); err == nil {
		f.Year = v
	}
	return f
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
	httphelper.Created(c, "Jadwal berhasil dibuat", data)
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
	httphelper.Success(c, "Jadwal berhasil diperbarui", data)
}

func (h *HandlerImpl) Publish(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var body schedule_dto.PublishRequest
	_ = c.ShouldBindJSON(&body)
	if err := h.svc.SetActive(c.Request.Context(), id, body.IsActive, appctx.UserID(c)); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Status jadwal diperbarui", nil)
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
	httphelper.Success(c, "Jadwal berhasil dihapus", nil)
}
