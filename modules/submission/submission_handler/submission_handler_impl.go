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
		UserID:                  appctx.UserID(c),
		OrganizationID:          appctx.OrganizationID(c),
		OrganizationTypeCode:    appctx.OrganizationTypeCode(c),
		WildcardTierAccess:      appctx.WildcardTierAccess(c),
		RequestedOrganizationID: requestedOrganizationID(c),
	}
}

// requestedOrganizationID membaca query `organizationID` opsional (target
// org-switcher di shell cms-ldk/cms-puskomda) — TIDAK divalidasi di sini,
// hanya diteruskan mentah; validasi accessible dilakukan di service lewat
// OrgScopeResolver.IsAccessible sebelum dipakai untuk menyaring data.
func requestedOrganizationID(c *gin.Context) *int64 {
	raw := c.Query("organizationID")
	if raw == "" {
		return nil
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return nil
	}
	return &id
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
	data, total, err := h.svc.List(c.Request.Context(), callerScope(c), q, c.Query("status"), c.Query("formCode"))
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

func (h *HandlerImpl) Review(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req submission_dto.ReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	data, err := h.svc.Review(c.Request.Context(), id, callerScope(c), req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Review berhasil disimpan", data)
}

func (h *HandlerImpl) EstablishLevel(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req submission_dto.EstablishLevelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	data, err := h.svc.EstablishLevel(c.Request.Context(), id, callerScope(c), req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Level berhasil ditetapkan", data)
}

func (h *HandlerImpl) Publish(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req submission_dto.VersionedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	data, err := h.svc.Publish(c.Request.Context(), id, callerScope(c), req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Hasil berhasil dipublikasikan", data)
}

func (h *HandlerImpl) Reopen(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req submission_dto.ReopenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	data, err := h.svc.Reopen(c.Request.Context(), id, callerScope(c), req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Pendataan dibuka kembali untuk koreksi", data)
}

func (h *HandlerImpl) Reassess(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req submission_dto.VersionedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	data, err := h.svc.Reassess(c.Request.Context(), id, callerScope(c), req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Reassessment berhasil diajukan", data)
}

func (h *HandlerImpl) ListKaders(c *gin.Context) {
	q := dto.ParseListQuery(c)
	data, total, err := h.svc.ListKaders(c.Request.Context(), callerScope(c), q, c.Query("status"))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", httphelper.BuildPagination(c, data, total, q.Page, q.Limit))
}

func (h *HandlerImpl) GetKaderCode(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	data, err := h.svc.GetKaderCode(c.Request.Context(), id, callerScope(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) DeactivateKader(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeactivateKader(c.Request.Context(), id, callerScope(c)); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Kader berhasil dinonaktifkan", nil)
}

func (h *HandlerImpl) ReassessKader(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req submission_dto.VersionedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	data, err := h.svc.ReassessKader(c.Request.Context(), id, callerScope(c), req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Pendataan dibuka kembali untuk diisi ulang", data)
}

func (h *HandlerImpl) ReinstateKader(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.ReinstateKader(c.Request.Context(), id, callerScope(c)); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Data kader berhasil diputihkan kembali", nil)
}
