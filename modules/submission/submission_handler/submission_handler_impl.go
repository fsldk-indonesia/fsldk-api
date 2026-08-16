package submission_handler

import (
	"strconv"

	"fsldk-api/base/appctx"
	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/modules/submission/submission_dto"
	"fsldk-api/modules/submission/submission_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct{ svc submission_service.Service }

// NewHandler membuat Handler submission.
func NewHandler(svc submission_service.Service) Handler { return &HandlerImpl{svc: svc} }

func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return id, true
}

func callerScope(c *gin.Context) submission_service.CallerScope {
	return submission_service.CallerScope{
		UserID:               appctx.UserID(c),
		OrganizationID:       appctx.OrganizationID(c),
		OrganizationTypeCode: appctx.OrganizationTypeCode(c),
		WildcardTierAccess:   appctx.WildcardTierAccess(c),
	}
}

func (h *HandlerImpl) Create(c *gin.Context) {
	var req submission_dto.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	data, err := h.svc.Create(c.Request.Context(), callerScope(c), req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Created(c, "Pendataan berhasil dibuat", data)
}

func (h *HandlerImpl) SaveAnswers(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req submission_dto.SaveAnswersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	data, err := h.svc.SaveAnswers(c.Request.Context(), id, callerScope(c), req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Jawaban berhasil disimpan", data)
}

func (h *HandlerImpl) Submit(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	data, err := h.svc.Submit(c.Request.Context(), id, callerScope(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Pendataan berhasil dikirim", data)
}

func (h *HandlerImpl) Cancel(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.Cancel(c.Request.Context(), id, callerScope(c)); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Pendataan berhasil dibatalkan", nil)
}

func (h *HandlerImpl) List(c *gin.Context) {
	q := dto.ParseListQuery(c)
	data, total, err := h.svc.List(c.Request.Context(), callerScope(c), q, c.Query("status"))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", httphelper.BuildPagination(c, data, total, q.Page, q.Limit))
}

func (h *HandlerImpl) Get(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	data, err := h.svc.Get(c.Request.Context(), id, callerScope(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}
