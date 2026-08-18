package organization_handler

import (
	"strconv"

	"fsldk-api/base/appctx"
	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/modules/organization/organization_dto"
	"fsldk-api/modules/organization/organization_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct{ svc organization_service.Service }

// NewHandler membuat Handler organization.
func NewHandler(svc organization_service.Service) Handler { return &HandlerImpl{svc: svc} }

func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return id, true
}

func callerScope(c *gin.Context) organization_service.CallerScope {
	return organization_service.CallerScope{
		UserID:               appctx.UserID(c),
		OrganizationID:       appctx.OrganizationID(c),
		OrganizationTypeCode: appctx.OrganizationTypeCode(c),
		WildcardTierAccess:   appctx.WildcardTierAccess(c),
	}
}

func (h *HandlerImpl) Me(c *gin.Context) {
	var siblingOf *int64
	if raw := c.Query("siblingOf"); raw != "" {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil && id > 0 {
			siblingOf = &id
		}
	}
	data, err := h.svc.AccessibleList(c.Request.Context(), callerScope(c), c.Query("organizationTypeCode"), siblingOf, c.Query("q"))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) Directory(c *gin.Context) {
	typeCode := c.Query("organizationTypeCode")
	if typeCode != "LDK" && typeCode != "PUSKOMDA" && typeCode != "PUSKOMNAS" {
		httphelper.Error(c, apperror.BadRequest("organizationTypeCode wajib diisi (LDK/PUSKOMDA/PUSKOMNAS)"))
		return
	}
	data, err := h.svc.Directory(c.Request.Context(), typeCode)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) List(c *gin.Context) {
	q := dto.ParseListQuery(c)
	data, total, err := h.svc.List(c.Request.Context(), callerScope(c), q, c.Query("organizationTypeCode"))
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
	data, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) Children(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	data, err := h.svc.Children(c.Request.Context(), id)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) Create(c *gin.Context) {
	var req organization_dto.CreateRequest
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
	httphelper.Created(c, "Organisasi berhasil dibuat", data)
}

func (h *HandlerImpl) Update(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req organization_dto.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	data, err := h.svc.Update(c.Request.Context(), id, req, appctx.UserID(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Organisasi berhasil diperbarui", data)
}

func (h *HandlerImpl) Deactivate(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.SetActive(c.Request.Context(), callerScope(c), id, false); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Organisasi berhasil dinonaktifkan", nil)
}

func (h *HandlerImpl) Reactivate(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.SetActive(c.Request.Context(), callerScope(c), id, true); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Organisasi berhasil diaktifkan kembali", nil)
}
