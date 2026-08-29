package campaign_handler

import (
	"strconv"

	"fsldk-api/base/appctx"
	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/modules/campaign/campaign_dto"
	"fsldk-api/modules/campaign/campaign_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct{ svc campaign_service.Service }

// NewHandler membuat Handler campaign.
func NewHandler(svc campaign_service.Service) Handler { return &HandlerImpl{svc: svc} }

func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return id, true
}

func callerScope(c *gin.Context) campaign_service.CallerScope {
	return campaign_service.CallerScope{
		UserID:               appctx.UserID(c),
		OrganizationID:       appctx.OrganizationID(c),
		OrganizationTypeCode: appctx.OrganizationTypeCode(c),
		WildcardTierAccess:   appctx.WildcardTierAccess(c),
	}
}

// ---------- Public ----------

func (h *HandlerImpl) PublicList(c *gin.Context) {
	q := dto.ParseListQuery(c)
	categoryID, _ := strconv.ParseInt(c.Query("categoryID"), 10, 64)
	data, total, err := h.svc.PublicList(c.Request.Context(), q, categoryID)
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

func (h *HandlerImpl) Categories(c *gin.Context) {
	data, err := h.svc.Categories(c.Request.Context())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

// ---------- Me ----------

func (h *HandlerImpl) MyList(c *gin.Context) {
	q := dto.ParseListQuery(c)
	data, total, err := h.svc.MyList(c.Request.Context(), callerScope(c), q)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", httphelper.BuildPagination(c, data, total, q.Page, q.Limit))
}

func (h *HandlerImpl) MyGet(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	data, err := h.svc.MyGet(c.Request.Context(), id, callerScope(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) bindCreate(c *gin.Context) (campaign_dto.CreateRequest, bool) {
	var req campaign_dto.CreateRequest
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

func (h *HandlerImpl) bindUpdate(c *gin.Context) (campaign_dto.UpdateRequest, bool) {
	var req campaign_dto.UpdateRequest
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
	req, ok := h.bindCreate(c)
	if !ok {
		return
	}
	data, err := h.svc.Create(c.Request.Context(), callerScope(c), req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Created(c, "Campaign berhasil dibuat", data)
}

func (h *HandlerImpl) Update(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	req, ok := h.bindUpdate(c)
	if !ok {
		return
	}
	data, err := h.svc.Update(c.Request.Context(), id, callerScope(c), req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Campaign berhasil diperbarui", data)
}

func (h *HandlerImpl) UpdateBeneficiary(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req campaign_dto.UpdateBeneficiaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	data, err := h.svc.UpdateBeneficiary(c.Request.Context(), id, callerScope(c), req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Rekening penerima berhasil diperbarui — berlaku setelah masa jeda keamanan 24 jam", data)
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
	httphelper.Success(c, "Campaign berhasil diajukan untuk moderasi", data)
}

// ---------- CMS ----------

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

func (h *HandlerImpl) ReviewHistory(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	data, err := h.svc.ReviewHistory(c.Request.Context(), id)
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
	var req campaign_dto.ReviewRequest
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
	httphelper.Success(c, "Review campaign berhasil disimpan", data)
}

func (h *HandlerImpl) Publish(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	data, err := h.svc.Publish(c.Request.Context(), id, callerScope(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Campaign berhasil dipublikasikan", data)
}

func (h *HandlerImpl) Pause(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	data, err := h.svc.Pause(c.Request.Context(), id, callerScope(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Campaign berhasil dijeda", data)
}

func (h *HandlerImpl) Resume(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	data, err := h.svc.Resume(c.Request.Context(), id, callerScope(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Campaign berhasil dilanjutkan", data)
}

func (h *HandlerImpl) Archive(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	data, err := h.svc.Archive(c.Request.Context(), id, callerScope(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Campaign berhasil diarsipkan", data)
}
