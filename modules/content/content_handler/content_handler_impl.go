package content_handler

import (
	"strconv"

	"fsldk-api/base/apperror"
	"fsldk-api/base/appctx"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/modules/content/content_dto"
	"fsldk-api/modules/content/content_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct{ svc content_service.Service }

// NewHandler membuat Handler content.
func NewHandler(svc content_service.Service) Handler { return &HandlerImpl{svc: svc} }

func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return id, true
}

func (h *HandlerImpl) PublicList(c *gin.Context) {
	data, err := h.svc.ListContent(c.Request.Context(), true)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) PublicByKey(c *gin.Context) {
	data, err := h.svc.GetContent(c.Request.Context(), c.Param("key"))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) PublicProfile(c *gin.Context) {
	data, err := h.svc.Profile(c.Request.Context())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) PublicOrg(c *gin.Context) {
	data, err := h.svc.ListOrg(c.Request.Context(), true)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) CMSList(c *gin.Context) {
	data, err := h.svc.ListContent(c.Request.Context(), false)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) Update(c *gin.Context) {
	var req content_dto.ContentUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	if err := h.svc.UpdateContent(c.Request.Context(), c.Param("key"), req, appctx.UserID(c)); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Konten berhasil diperbarui", nil)
}

func (h *HandlerImpl) OrgList(c *gin.Context) {
	data, err := h.svc.ListOrg(c.Request.Context(), false)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) bindOrg(c *gin.Context) (content_dto.OrgRequest, bool) {
	var req content_dto.OrgRequest
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

func (h *HandlerImpl) OrgCreate(c *gin.Context) {
	req, ok := h.bindOrg(c)
	if !ok {
		return
	}
	id, err := h.svc.CreateOrg(c.Request.Context(), req, appctx.UserID(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Created(c, "Pengurus berhasil ditambahkan", gin.H{"structureID": id})
}

func (h *HandlerImpl) OrgUpdate(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	req, ok := h.bindOrg(c)
	if !ok {
		return
	}
	if err := h.svc.UpdateOrg(c.Request.Context(), id, req); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Pengurus berhasil diperbarui", nil)
}

func (h *HandlerImpl) OrgDelete(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteOrg(c.Request.Context(), id); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Pengurus berhasil dihapus", nil)
}
