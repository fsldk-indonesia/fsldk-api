package dynamicform_handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"strconv"
	"strings"

	"fsldk-api/base/appctx"
	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/constants"
	"fsldk-api/modules/dynamicform/dynamicform_dto"
	"fsldk-api/modules/dynamicform/dynamicform_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl is the Handler implementation.
type HandlerImpl struct{ svc dynamicform_service.Service }

// NewHandler creates the dynamicform Handler.
func NewHandler(svc dynamicform_service.Service) Handler { return &HandlerImpl{svc: svc} }

// ---------------------------------------------------------------------------
// shared helpers
// ---------------------------------------------------------------------------

func idParam(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return id, true
}

func permsOf(c *gin.Context) []string {
	if v, ok := c.Get(constants.CtxPermissions); ok {
		if list, ok := v.([]string); ok {
			return list
		}
	}
	return nil
}

// optionalUser returns the caller's id/email when a valid token was present
// (routes use OptionalAuth), else nil/nil.
func optionalUser(c *gin.Context) (*int64, *string) {
	id := appctx.UserID(c)
	if id == 0 {
		return nil, nil
	}
	email := appctx.Email(c)
	return &id, &email
}

func bindForm(c *gin.Context) (dynamicform_dto.FormRequest, bool) {
	var req dynamicform_dto.FormRequest
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

func bindField(c *gin.Context) (dynamicform_dto.FieldRequest, bool) {
	var req dynamicform_dto.FieldRequest
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

// multipartValues reads every "field_<id>" form value into a map, plus the raw
// value map for extra keys like _hp_website / _form_ts.
func multipartValues(c *gin.Context) (values map[int64][]string, files map[int64]*multipart.FileHeader, raw map[string][]string) {
	values = map[int64][]string{}
	files = map[int64]*multipart.FileHeader{}
	raw = map[string][]string{}

	if form, err := c.MultipartForm(); err == nil && form != nil {
		for k, v := range form.Value {
			raw[k] = v
			if id, ok := fieldKeyID(k); ok {
				values[id] = v
			}
		}
		for k, fhs := range form.File {
			if len(fhs) == 0 {
				continue
			}
			if id, ok := fieldKeyID(k); ok {
				files[id] = fhs[0]
			}
		}
		return
	}
	// Fallback: urlencoded body.
	_ = c.Request.ParseForm()
	for k, v := range c.Request.PostForm {
		raw[k] = v
		if id, ok := fieldKeyID(k); ok {
			values[id] = v
		}
	}
	return
}

func fieldKeyID(key string) (int64, bool) {
	if !strings.HasPrefix(key, "field_") {
		return 0, false
	}
	id, err := strconv.ParseInt(strings.TrimPrefix(key, "field_"), 10, 64)
	return id, err == nil
}

func firstRaw(raw map[string][]string, key string) string {
	if v, ok := raw[key]; ok && len(v) > 0 {
		return v[0]
	}
	return ""
}

// ---------------------------------------------------------------------------
// public
// ---------------------------------------------------------------------------

func (h *HandlerImpl) PublicGet(c *gin.Context) {
	uid, email := optionalUser(c)
	data, err := h.svc.GetPublicForm(c.Request.Context(), c.Param("slug"), uid, email)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) PublicSubmit(c *gin.Context) {
	values, files, raw := multipartValues(c)
	formTS, _ := strconv.ParseInt(firstRaw(raw, "_form_ts"), 10, 64)
	uid, email := optionalUser(c)

	in := dynamicform_service.SubmitInput{
		Values: values, Files: files,
		IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		Honeypot: firstRaw(raw, "_hp_website"), FormTS: formTS,
		AuthUserID: uid, AuthUserEmail: email,
	}
	res, err := h.svc.Submit(c.Request.Context(), c.Param("slug"), in)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Tanggapan Anda telah terkirim", res)
}

func (h *HandlerImpl) SaveDraft(c *gin.Context) {
	var req dynamicform_dto.DraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := h.svc.SaveDraft(c.Request.Context(), c.Param("slug"), appctx.UserID(c), req.Answers); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", gin.H{"success": true})
}

func (h *HandlerImpl) StageDraftFile(c *gin.Context) {
	fieldID, ok := idParam(c, "fieldID")
	if !ok {
		return
	}
	fh, err := c.FormFile("file")
	if err != nil {
		httphelper.Error(c, apperror.BadRequest("Berkas tidak ditemukan"))
		return
	}
	name, err := h.svc.StageDraftFile(c.Request.Context(), c.Param("slug"), appctx.UserID(c), fieldID, fh)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", gin.H{"success": true, "fieldID": fieldID, "originalFileName": name})
}

func (h *HandlerImpl) RemoveDraftFile(c *gin.Context) {
	fieldID, ok := idParam(c, "fieldID")
	if !ok {
		return
	}
	if err := h.svc.RemoveDraftFile(c.Request.Context(), c.Param("slug"), appctx.UserID(c), fieldID); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", gin.H{"success": true})
}

// ---------------------------------------------------------------------------
// CMS — forms
// ---------------------------------------------------------------------------

func (h *HandlerImpl) CMSList(c *gin.Context) {
	q := dto.ParseListQuery(c)
	f := dynamicform_dto.FormFilter{
		Status:   strings.TrimSpace(c.Query("status")),
		DateFrom: strings.TrimSpace(c.Query("dateFrom")),
		DateTo:   strings.TrimSpace(c.Query("dateTo")),
	}
	data, total, err := h.svc.ListForms(c.Request.Context(), q, f, appctx.UserID(c), permsOf(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", httphelper.BuildPagination(c, data, total, q.Page, q.Limit))
}

func (h *HandlerImpl) CMSGet(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	data, err := h.svc.GetForm(c.Request.Context(), id, appctx.UserID(c), permsOf(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) Create(c *gin.Context) {
	req, ok := bindForm(c)
	if !ok {
		return
	}
	data, err := h.svc.CreateForm(c.Request.Context(), req, appctx.UserID(c), permsOf(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Created(c, "Formulir berhasil dibuat", data)
}

func (h *HandlerImpl) Update(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	req, ok := bindForm(c)
	if !ok {
		return
	}
	data, err := h.svc.UpdateForm(c.Request.Context(), id, req, appctx.UserID(c), permsOf(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Formulir berhasil diperbarui", data)
}

func (h *HandlerImpl) SetStatus(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	var req dynamicform_dto.StatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	if err := h.svc.SetStatus(c.Request.Context(), id, req.Status, appctx.UserID(c), permsOf(c)); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Status formulir diperbarui", nil)
}

func (h *HandlerImpl) Delete(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteForm(c.Request.Context(), id, appctx.UserID(c), permsOf(c)); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Formulir berhasil dihapus", nil)
}

func (h *HandlerImpl) BulkDelete(c *gin.Context) {
	var req dynamicform_dto.BulkDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	res, err := h.svc.BulkDelete(c.Request.Context(), req.IDs, appctx.UserID(c), permsOf(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Formulir terpilih diproses", res)
}

// ---------------------------------------------------------------------------
// CMS — builder
// ---------------------------------------------------------------------------

func (h *HandlerImpl) AddField(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	req, ok := bindField(c)
	if !ok {
		return
	}
	data, err := h.svc.AddField(c.Request.Context(), id, req, appctx.UserID(c), permsOf(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Created(c, "Field ditambahkan", data)
}

func (h *HandlerImpl) UpdateField(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	fieldID, ok := idParam(c, "fieldID")
	if !ok {
		return
	}
	req, ok := bindField(c)
	if !ok {
		return
	}
	data, err := h.svc.UpdateField(c.Request.Context(), id, fieldID, req, appctx.UserID(c), permsOf(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Field diperbarui", data)
}

func (h *HandlerImpl) RemoveField(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	fieldID, ok := idParam(c, "fieldID")
	if !ok {
		return
	}
	if err := h.svc.RemoveField(c.Request.Context(), id, fieldID, appctx.UserID(c), permsOf(c)); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Field dihapus", nil)
}

func (h *HandlerImpl) ReorderFields(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	var req dynamicform_dto.ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	if err := h.svc.ReorderFields(c.Request.Context(), id, req.Order, appctx.UserID(c), permsOf(c)); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Urutan field disimpan", nil)
}

// ---------------------------------------------------------------------------
// CMS — rekap / analytics / export
// ---------------------------------------------------------------------------

func (h *HandlerImpl) ListSubmissions(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	q := dto.ParseListQuery(c)
	f := dynamicform_dto.SubmissionFilter{
		ValidOnly: c.Query("validOnly") == "true",
		DateFrom:  strings.TrimSpace(c.Query("dateFrom")),
		DateTo:    strings.TrimSpace(c.Query("dateTo")),
	}
	data, total, err := h.svc.ListSubmissions(c.Request.Context(), id, q, f, appctx.UserID(c), permsOf(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", httphelper.BuildPagination(c, data, total, q.Page, q.Limit))
}

func (h *HandlerImpl) GetSubmission(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	subID, ok := idParam(c, "subID")
	if !ok {
		return
	}
	data, err := h.svc.GetSubmission(c.Request.Context(), id, subID, appctx.UserID(c), permsOf(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) UpdateSubmission(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	subID, ok := idParam(c, "subID")
	if !ok {
		return
	}
	values, files, _ := multipartValues(c)
	req := dynamicform_dto.EditSubmissionRequest{Answers: map[string]json.RawMessage{}}
	for fieldID, v := range values {
		key := "field_" + strconv.FormatInt(fieldID, 10)
		if len(v) > 1 {
			b, _ := json.Marshal(v)
			req.Answers[key] = b
			continue
		}
		req.Answers[key] = json.RawMessage(v[0])
	}
	if err := h.svc.UpdateSubmission(c.Request.Context(), id, subID, appctx.UserID(c), permsOf(c), req, files); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Tanggapan diperbarui", nil)
}

func (h *HandlerImpl) DeleteSubmission(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	subID, ok := idParam(c, "subID")
	if !ok {
		return
	}
	if err := h.svc.DeleteSubmission(c.Request.Context(), id, subID, appctx.UserID(c), permsOf(c)); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Tanggapan dihapus", nil)
}

func (h *HandlerImpl) ExportCSV(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	var buf bytes.Buffer
	slug, err := h.svc.ExportCSV(c.Request.Context(), id, appctx.UserID(c), permsOf(c), &buf)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-responses.csv"`, slug))
	c.String(200, buf.String())
}

func (h *HandlerImpl) DeleteResponses(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteResponses(c.Request.Context(), id, appctx.UserID(c), permsOf(c)); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Semua tanggapan dihapus", nil)
}

func (h *HandlerImpl) Analytics(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	data, err := h.svc.GetAnalytics(c.Request.Context(), id, appctx.UserID(c), permsOf(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

// ---------------------------------------------------------------------------
// CMS — Google Sheets
// ---------------------------------------------------------------------------

func (h *HandlerImpl) GSheetConnect(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	data, err := h.svc.GSheetConnect(c.Request.Context(), id, appctx.UserID(c), permsOf(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Google Sheet terhubung", data)
}

func (h *HandlerImpl) GSheetResync(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	data, err := h.svc.GSheetResync(c.Request.Context(), id, appctx.UserID(c), permsOf(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Sinkronisasi ulang dijadwalkan", data)
}

func (h *HandlerImpl) GSheetDisconnect(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		return
	}
	data, err := h.svc.GSheetDisconnect(c.Request.Context(), id, appctx.UserID(c), permsOf(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Google Sheet diputuskan", data)
}
