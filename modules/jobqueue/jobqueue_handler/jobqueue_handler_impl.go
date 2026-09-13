package jobqueue_handler

import (
	"strconv"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/modules/jobqueue/jobqueue_dto"
	"fsldk-api/modules/jobqueue/jobqueue_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct{ svc jobqueue_service.Service }

// NewHandler membuat Handler job queue.
func NewHandler(svc jobqueue_service.Service) Handler { return &HandlerImpl{svc: svc} }

func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return id, true
}

func (h *HandlerImpl) CMSList(c *gin.Context) {
	q := dto.ParseListQuery(c)
	statuses := dto.ParseCSV(c.Query("status"))
	queues := dto.ParseCSV(c.Query("queue"))
	dateFrom := c.Query("dateFrom")
	dateTo := c.Query("dateTo")
	data, total, err := h.svc.CMSList(c.Request.Context(), q, statuses, queues, dateFrom, dateTo)
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
	res, err := h.svc.CMSGet(c.Request.Context(), id)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", res)
}

func (h *HandlerImpl) CMSStats(c *gin.Context) {
	res, err := h.svc.CMSStats(c.Request.Context())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", res)
}

func (h *HandlerImpl) Retry(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.Retry(c.Request.Context(), id); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Job akan dicoba ulang", nil)
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
	httphelper.Success(c, "Job dihapus", nil)
}

func (h *HandlerImpl) BulkDelete(c *gin.Context) {
	var req jobqueue_dto.BulkDeleteRequest
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
	httphelper.Success(c, "Job terpilih berhasil dihapus", nil)
}
