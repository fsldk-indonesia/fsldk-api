package submission_form_handler

import (
	"errors"
	"io"
	"strconv"

	"fsldk-api/base/appctx"
	"fsldk-api/base/apperror"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/modules/submission_form/submission_form_dto"
	"fsldk-api/modules/submission_form/submission_form_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct {
	svc submission_form_service.Service
}

// NewHandler membuat Handler submission_form.
func NewHandler(svc submission_form_service.Service) Handler { return &HandlerImpl{svc: svc} }

func idFromParam(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return id, true
}

func (h *HandlerImpl) ListForms(c *gin.Context) {
	data, err := h.svc.ListForms(c.Request.Context())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) CreateForm(c *gin.Context) {
	var req submission_form_dto.CreateFormRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	data, err := h.svc.CreateForm(c.Request.Context(), req, appctx.UserID(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Created(c, "Form berhasil dibuat", data)
}

func (h *HandlerImpl) GetForm(c *gin.Context) {
	id, ok := idFromParam(c, "formID")
	if !ok {
		return
	}
	data, err := h.svc.GetForm(c.Request.Context(), id)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) CreateVersion(c *gin.Context) {
	formID, ok := idFromParam(c, "formID")
	if !ok {
		return
	}
	// Body opsional — membuat version kosong tidak memerlukan payload apapun.
	var req submission_form_dto.CreateVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	data, err := h.svc.CreateVersion(c.Request.Context(), formID, req, appctx.UserID(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Created(c, "Version baru berhasil dibuat", data)
}

func (h *HandlerImpl) GetVersion(c *gin.Context) {
	id, ok := idFromParam(c, "versionID")
	if !ok {
		return
	}
	data, err := h.svc.GetVersion(c.Request.Context(), id)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) PublishVersion(c *gin.Context) {
	id, ok := idFromParam(c, "versionID")
	if !ok {
		return
	}
	data, err := h.svc.PublishVersion(c.Request.Context(), id, appctx.UserID(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Version berhasil dipublish", data)
}

func (h *HandlerImpl) CreateSection(c *gin.Context) {
	versionID, ok := idFromParam(c, "versionID")
	if !ok {
		return
	}
	var req submission_form_dto.CreateSectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	data, err := h.svc.CreateSection(c.Request.Context(), versionID, req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Created(c, "Section berhasil ditambahkan", data)
}

func (h *HandlerImpl) UpdateSection(c *gin.Context) {
	id, ok := idFromParam(c, "sectionID")
	if !ok {
		return
	}
	var req submission_form_dto.UpdateSectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	data, err := h.svc.UpdateSection(c.Request.Context(), id, req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Section berhasil diperbarui", data)
}

func (h *HandlerImpl) DeleteSection(c *gin.Context) {
	id, ok := idFromParam(c, "sectionID")
	if !ok {
		return
	}
	if err := h.svc.DeleteSection(c.Request.Context(), id); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Section berhasil dihapus", nil)
}

func (h *HandlerImpl) CreateField(c *gin.Context) {
	sectionID, ok := idFromParam(c, "sectionID")
	if !ok {
		return
	}
	var req submission_form_dto.CreateFieldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	data, err := h.svc.CreateField(c.Request.Context(), sectionID, req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Created(c, "Field berhasil ditambahkan", data)
}

func (h *HandlerImpl) UpdateField(c *gin.Context) {
	id, ok := idFromParam(c, "fieldID")
	if !ok {
		return
	}
	var req submission_form_dto.UpdateFieldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	data, err := h.svc.UpdateField(c.Request.Context(), id, req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Field berhasil diperbarui", data)
}

func (h *HandlerImpl) DeleteField(c *gin.Context) {
	id, ok := idFromParam(c, "fieldID")
	if !ok {
		return
	}
	if err := h.svc.DeleteField(c.Request.Context(), id); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Field berhasil dihapus", nil)
}

func (h *HandlerImpl) CreateOption(c *gin.Context) {
	fieldID, ok := idFromParam(c, "fieldID")
	if !ok {
		return
	}
	var req submission_form_dto.CreateOptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	data, err := h.svc.CreateOption(c.Request.Context(), fieldID, req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Created(c, "Pilihan berhasil ditambahkan", data)
}

func (h *HandlerImpl) UpdateOption(c *gin.Context) {
	id, ok := idFromParam(c, "optionID")
	if !ok {
		return
	}
	var req submission_form_dto.UpdateOptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	data, err := h.svc.UpdateOption(c.Request.Context(), id, req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Pilihan berhasil diperbarui", data)
}

func (h *HandlerImpl) DeleteOption(c *gin.Context) {
	id, ok := idFromParam(c, "optionID")
	if !ok {
		return
	}
	if err := h.svc.DeleteOption(c.Request.Context(), id); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Pilihan berhasil dihapus", nil)
}
