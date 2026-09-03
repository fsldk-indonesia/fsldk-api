package dynamicform_service

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"slices"
	"strings"
	"time"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/slug"
	"fsldk-api/constants"
	"fsldk-api/modules/dynamicform/dynamicform_dto"
	"fsldk-api/modules/dynamicform/dynamicform_model"
	"fsldk-api/modules/dynamicform/dynamicform_repository"
	"fsldk-api/modules/jobqueue/jobqueue_dto"
	"fsldk-api/pkg/auditlog"
	"fsldk-api/pkg/gsheet"
	"fsldk-api/pkg/mailer"
)

const (
	auditEntity   = "dynamic_form"
	sysEmailLabel = "Alamat Email"
	timeLayout    = "2006-01-02 15:04:05"
)

// Uploader is the slice of pkg/upload.Uploader the service needs.
type Uploader interface {
	SaveDocument(fh *multipart.FileHeader) (string, error)
	SaveImage(fh *multipart.FileHeader) (string, error)
	DeleteFile(publicURL string) error
	FileSize(publicURL string) (int64, error)
	LocalPath(publicURL string) string
}

// FormMailer is the mailer method this service uses.
type FormMailer interface {
	SendFormSubmissionConfirmation(toEmail, toName, formTitle string, answers []mailer.AnswerPair, submittedAt string) error
}

// JobEnqueuer enqueues background jobs (implemented by jobqueue_service).
type JobEnqueuer interface {
	Enqueue(ctx context.Context, in jobqueue_dto.EnqueueInput) (int64, error)
}

// ServiceImpl is the Service implementation.
type ServiceImpl struct {
	repo           dynamicform_repository.Repository
	uploader       Uploader
	mailer         FormMailer
	audit          *auditlog.Logger
	gsheet         gsheet.Client
	jobs           JobEnqueuer
	frontendURL    string
	gsheetFolderID string
}

// NewService creates the dynamicform Service.
func NewService(
	repo dynamicform_repository.Repository,
	uploader Uploader,
	mailer FormMailer,
	audit *auditlog.Logger,
	gs gsheet.Client,
	jobs JobEnqueuer,
	frontendURL, gsheetFolderID string,
) Service {
	return &ServiceImpl{
		repo: repo, uploader: uploader, mailer: mailer, audit: audit,
		gsheet: gs, jobs: jobs, frontendURL: strings.TrimRight(frontendURL, "/"),
		gsheetFolderID: gsheetFolderID,
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func fmtTimePtr(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format(timeLayout)
}

func parseTimePtr(s *string) *time.Time {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil
	}
	t, err := time.ParseInLocation(timeLayout, strings.TrimSpace(*s), time.Local)
	if err != nil {
		return nil
	}
	return &t
}

func strOr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func rawJSON(s *string) json.RawMessage {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil
	}
	return json.RawMessage(*s)
}

func marshalOrNil(v any) *string {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	s := string(b)
	return &s
}

// getOwnedForm loads a form and enforces the ownership guard: creator, holder
// of dynamicform.manage.all, or an editor/manager collaborator.
func (s *ServiceImpl) getOwnedForm(ctx context.Context, id, actorID int64, perms []string) (dynamicform_model.Form, error) {
	form, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dynamicform_model.Form{}, apperror.NotFound("Formulir tidak ditemukan")
	}
	if form.CreatedBy != nil && *form.CreatedBy == actorID {
		return form, nil
	}
	if slices.Contains(perms, constants.PermDynamicFormManageAll) {
		return form, nil
	}
	isCollab, _ := s.repo.IsCollaborator(ctx, id, actorID, "editor", "manager")
	if isCollab {
		return form, nil
	}
	return dynamicform_model.Form{}, apperror.Forbidden("Anda tidak memiliki akses ke formulir ini")
}

func (s *ServiceImpl) logAudit(ctx context.Context, actorID, formID int64, action string, before, after, meta any) {
	if s.audit == nil {
		return
	}
	s.audit.LogForm(ctx, auditlog.Entry{
		ActorUserID: actorID, Action: action, Entity: auditEntity, EntityID: formID,
		Before: before, After: after, Metadata: meta,
	})
}

func (s *ServiceImpl) uniqueSlug(ctx context.Context, title string, exceptID int64) (string, error) {
	base := slug.Make(title)
	candidate := base
	for i := 2; i < 200; i++ {
		exists, err := s.repo.SlugExists(ctx, candidate, exceptID)
		if err != nil {
			return "", apperror.Internal("")
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
	return fmt.Sprintf("%s-%d", base, time.Now().UnixNano()), nil
}

func notifyEmailsOf(form dynamicform_model.Form) []string {
	if form.NotifyEmailsJSON == nil || strings.TrimSpace(*form.NotifyEmailsJSON) == "" {
		return nil
	}
	var out []string
	_ = json.Unmarshal([]byte(*form.NotifyEmailsJSON), &out)
	return out
}

// ---------------------------------------------------------------------------
// response mapping
// ---------------------------------------------------------------------------

func toFieldResponse(f dynamicform_model.Field) dynamicform_dto.FieldResponse {
	return dynamicform_dto.FieldResponse{
		FieldID: f.FieldID, FormID: f.FormID, SectionID: f.SectionID, FieldType: f.FieldType,
		Label: f.Label, Placeholder: f.Placeholder, HelpText: f.HelpText, IsRequired: f.IsRequired,
		IsSystemField: f.IsSystemField, SortOrder: f.SortOrder,
		Options: rawJSON(f.OptionsJSON), Validation: rawJSON(f.ValidationJSON),
		DefaultValue: f.DefaultValue, ConditionalLogic: rawJSON(f.ConditionalLogicJSON),
		FieldConfig: rawJSON(f.FieldConfigJSON),
	}
}

func toSectionResponse(sec dynamicform_model.Section) dynamicform_dto.SectionResponse {
	return dynamicform_dto.SectionResponse{
		SectionID: sec.SectionID, FormID: sec.FormID, Title: sec.Title,
		Description: sec.Description, SortOrder: sec.SortOrder,
	}
}

func (s *ServiceImpl) toFormResponse(form dynamicform_model.Form, fields []dynamicform_model.Field, sections []dynamicform_model.Section, collabs []dynamicform_model.Collaborator, fieldCount int) dynamicform_dto.FormResponse {
	fr := dynamicform_dto.FormResponse{
		FormID: form.FormID, Title: form.Title, Slug: form.Slug, Description: strOr(form.Description),
		Status: form.Status, Version: form.Version, MaxSubmission: form.MaxSubmission,
		IsMultipleSubmit: form.IsMultipleSubmit, RequireLogin: form.RequireLogin,
		StartDate: fmtTimePtr(form.StartDate), EndDate: fmtTimePtr(form.EndDate),
		ConfirmationMessage: strOr(form.ConfirmationMessage), RedirectURL: strOr(form.RedirectURL),
		NotifyEmails: notifyEmailsOf(form), SendConfirmationEmail: form.SendConfirmationEmail,
		RateLimitPerIP: form.RateLimitPerIP, RateLimitWindowMinutes: form.RateLimitWindowMinutes,
		GsheetEnabled: form.GsheetEnabled, GsheetSpreadsheetURL: strOr(form.GsheetSpreadsheetURL),
		GsheetLastSyncDate: fmtTimePtr(form.GsheetLastSyncDate), GsheetLastSyncError: strOr(form.GsheetLastSyncError),
		TotalSubmission: form.TotalSubmission, IsActive: form.IsActive,
		CreatedDate: form.CreatedDate.Format(timeLayout), CreatorName: form.CreatorName,
		UpdatedDate: fmtTimePtr(form.UpdatedDate), FieldCount: fieldCount,
		PublicURL: s.frontendURL + "/form/" + form.Slug,
	}
	for _, f := range fields {
		fr.Fields = append(fr.Fields, toFieldResponse(f))
	}
	for _, sec := range sections {
		fr.Sections = append(fr.Sections, toSectionResponse(sec))
	}
	for _, c := range collabs {
		fr.Collaborators = append(fr.Collaborators, dynamicform_dto.CollaboratorResponse{
			UserID: c.UserID, Role: c.Role, UserName: c.UserName, UserEmail: c.UserEmail,
		})
	}
	return fr
}

// ---------------------------------------------------------------------------
// CMS: forms
// ---------------------------------------------------------------------------

func (s *ServiceImpl) formValuesFromRequest(req dynamicform_dto.FormRequest) map[string]any {
	rateLimit := req.RateLimitPerIP
	if rateLimit <= 0 {
		rateLimit = 5
	}
	window := req.RateLimitWindowMinutes
	if window <= 0 {
		window = 10
	}
	return map[string]any{
		"title":                  strings.TrimSpace(req.Title),
		"description":            req.Description,
		"maxSubmission":          req.MaxSubmission,
		"isMultipleSubmit":       req.IsMultipleSubmit,
		"requireLogin":           req.RequireLogin,
		"startDate":              parseTimePtr(req.StartDate),
		"endDate":                parseTimePtr(req.EndDate),
		"confirmationMessage":    req.ConfirmationMessage,
		"redirectUrl":            req.RedirectURL,
		"notifyEmailsJSON":       marshalOrNil(req.NotifyEmails),
		"sendConfirmationEmail":  req.SendConfirmationEmail,
		"rateLimitPerIP":         rateLimit,
		"rateLimitWindowMinutes": window,
		"gsheetEnabled":          req.GsheetEnabled,
	}
}

func (s *ServiceImpl) CreateForm(ctx context.Context, req dynamicform_dto.FormRequest, actorID int64, perms []string) (dynamicform_dto.FormResponse, error) {
	slugStr, err := s.uniqueSlug(ctx, req.Title, 0)
	if err != nil {
		return dynamicform_dto.FormResponse{}, err
	}
	values := s.formValuesFromRequest(req)
	values["slug"] = slugStr
	values["status"] = constants.DynamicFormStatusDraft
	values["version"] = 1
	values["gsheetTabName"] = "Responses"
	values["totalSubmission"] = 0
	values["isActive"] = 1
	values["createdDate"] = time.Now()
	values["createdBy"] = actorID

	id, err := s.repo.CreateForm(ctx, values)
	if err != nil {
		return dynamicform_dto.FormResponse{}, apperror.Internal("Gagal menyimpan formulir")
	}

	// Auto system email field — cannot be deleted or retyped.
	helpText := "Email akan digunakan untuk konfirmasi pengisian."
	if _, err := s.repo.AddField(ctx, id, map[string]any{
		"fieldType": "email", "label": sysEmailLabel, "isRequired": 1, "isSystemField": 1,
		"helpText": helpText, "isActive": 1,
	}); err != nil {
		return dynamicform_dto.FormResponse{}, apperror.Internal("Gagal menambahkan field email sistem")
	}

	if len(req.Collaborators) > 0 {
		_ = s.repo.ReplaceCollaborators(ctx, id, req.Collaborators)
	}
	s.logAudit(ctx, actorID, id, "create", nil, values, nil)
	return s.GetForm(ctx, id, actorID, perms)
}

func (s *ServiceImpl) UpdateForm(ctx context.Context, id int64, req dynamicform_dto.FormRequest, actorID int64, perms []string) (dynamicform_dto.FormResponse, error) {
	form, err := s.getOwnedForm(ctx, id, actorID, perms)
	if err != nil {
		return dynamicform_dto.FormResponse{}, err
	}
	values := s.formValuesFromRequest(req)
	values["version"] = form.Version + 1
	values["updatedBy"] = actorID
	values["updatedDate"] = time.Now()
	if err := s.repo.UpdateForm(ctx, id, values); err != nil {
		return dynamicform_dto.FormResponse{}, apperror.Internal("")
	}
	_ = s.repo.ReplaceCollaborators(ctx, id, req.Collaborators)
	s.logAudit(ctx, actorID, id, "update", form, values, nil)

	// gsheet toggle false -> true: create the sheet now (best-effort).
	if req.GsheetEnabled && !form.GsheetEnabled && form.GsheetSpreadsheetID == nil && s.gsheet.Enabled() {
		if refreshed, gErr := s.repo.GetByID(ctx, id); gErr == nil {
			_ = s.ensureSheet(ctx, refreshed)
		}
	}
	return s.GetForm(ctx, id, actorID, perms)
}

func (s *ServiceImpl) SetStatus(ctx context.Context, id int64, status string, actorID int64, perms []string) error {
	form, err := s.getOwnedForm(ctx, id, actorID, perms)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateForm(ctx, id, map[string]any{
		"status": status, "updatedBy": actorID, "updatedDate": time.Now(),
	}); err != nil {
		return apperror.Internal("")
	}
	s.logAudit(ctx, actorID, id, statusAction(status), map[string]string{"status": form.Status}, map[string]string{"status": status}, nil)

	if status == constants.DynamicFormStatusPublished && form.GsheetEnabled && form.GsheetSpreadsheetID == nil && s.gsheet.Enabled() {
		if refreshed, gErr := s.repo.GetByID(ctx, id); gErr == nil {
			_ = s.ensureSheet(ctx, refreshed)
		}
	}
	return nil
}

func statusAction(status string) string {
	switch status {
	case constants.DynamicFormStatusPublished:
		return "publish"
	case constants.DynamicFormStatusClosed:
		return "close"
	case constants.DynamicFormStatusArchived:
		return "archive"
	default:
		return "update"
	}
}

func (s *ServiceImpl) DeleteForm(ctx context.Context, id int64, actorID int64, perms []string) error {
	if _, err := s.getOwnedForm(ctx, id, actorID, perms); err != nil {
		return err
	}
	urls, err := s.repo.PurgeFormChildren(ctx, id)
	if err != nil {
		return apperror.Internal("")
	}
	if err := s.repo.SoftDeleteForm(ctx, id); err != nil {
		return apperror.Internal("")
	}
	for _, u := range urls {
		_ = s.uploader.DeleteFile(u)
	}
	s.logAudit(ctx, actorID, id, "delete", nil, nil, nil)
	return nil
}

func (s *ServiceImpl) BulkDelete(ctx context.Context, ids []int64, actorID int64, perms []string) (dynamicform_dto.BulkDeleteResult, error) {
	res := dynamicform_dto.BulkDeleteResult{Deleted: []int64{}, Skipped: []int64{}}
	for _, id := range ids {
		if err := s.DeleteForm(ctx, id, actorID, perms); err != nil {
			res.Skipped = append(res.Skipped, id)
			continue
		}
		res.Deleted = append(res.Deleted, id)
	}
	return res, nil
}

var formSortColumns = map[string]string{
	"title":           "f.title",
	"status":          "f.status",
	"totalSubmission": "f.totalSubmission",
	"createdDate":     "f.createdDate",
}

func (s *ServiceImpl) ListForms(ctx context.Context, q dto.ListQuery, f dynamicform_dto.FormFilter, actorID int64, perms []string) ([]dynamicform_dto.FormResponse, int, error) {
	f.Limit = q.Limit
	f.Offset = q.Offset()
	f.OrderBy = q.OrderBy(formSortColumns, "f.createdDate DESC")
	if f.Search == "" {
		f.Search = q.Search
	}
	f.ActorID = actorID
	f.MineOnly = !slices.Contains(perms, constants.PermDynamicFormManageAll)

	forms, total, counts, err := s.repo.ListForms(ctx, f)
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	out := make([]dynamicform_dto.FormResponse, 0, len(forms))
	for i, form := range forms {
		fc := 0
		if i < len(counts) {
			fc = counts[i]
		}
		out = append(out, s.toFormResponse(form, nil, nil, nil, fc))
	}
	return out, int(total), nil
}

func (s *ServiceImpl) GetForm(ctx context.Context, id int64, actorID int64, perms []string) (dynamicform_dto.FormResponse, error) {
	form, err := s.getOwnedForm(ctx, id, actorID, perms)
	if err != nil {
		return dynamicform_dto.FormResponse{}, err
	}
	fields, _ := s.repo.ListFields(ctx, id, true)
	sections, _ := s.repo.ListSections(ctx, id)
	collabs, _ := s.repo.ListCollaborators(ctx, id)
	fieldCount := 0
	for _, fld := range fields {
		if !isDisplayType(fld.FieldType) {
			fieldCount++
		}
	}
	return s.toFormResponse(form, fields, sections, collabs, fieldCount), nil
}

// ---------------------------------------------------------------------------
// Builder: fields
// ---------------------------------------------------------------------------

func choiceType(t string) bool { return t == "dropdown" || t == "radio" || t == "checkbox" }

// fieldValuesFromRequest normalises a FieldRequest into a column map. It loosens
// `label` for section_break/image and requires options for choice fields.
func (s *ServiceImpl) fieldValuesFromRequest(req dynamicform_dto.FieldRequest) (map[string]any, error) {
	label := strings.TrimSpace(req.Label)
	if label == "" && req.FieldType != "section_break" && req.FieldType != "image" {
		return nil, apperror.BadRequest("Label field wajib diisi")
	}
	if choiceType(req.FieldType) && len(req.Options) == 0 {
		return nil, apperror.BadRequest("Field pilihan wajib memiliki minimal 1 opsi")
	}
	values := map[string]any{
		"fieldType":            req.FieldType,
		"label":                label,
		"placeholder":          req.Placeholder,
		"isRequired":           req.IsRequired,
		"defaultValue":         req.DefaultValue,
		"optionsJSON":          marshalOrNil(req.Options),
		"validationJSON":       marshalOrNil(req.Validation),
		"conditionalLogicJSON": rawStringOrNil(req.ConditionalLogic),
		"fieldConfigJSON":      rawStringOrNil(req.FieldConfig),
	}
	if len(req.Options) == 0 {
		values["optionsJSON"] = nil
	}
	// For image fields, helpText stores the image URL (reference pattern).
	if req.FieldType == "image" && req.ImageURL != nil {
		values["helpText"] = *req.ImageURL
	} else {
		values["helpText"] = req.HelpText
	}
	return values, nil
}

func rawStringOrNil(r json.RawMessage) *string {
	if len(r) == 0 || string(r) == "null" {
		return nil
	}
	s := string(r)
	return &s
}

func (s *ServiceImpl) AddField(ctx context.Context, formID int64, req dynamicform_dto.FieldRequest, actorID int64, perms []string) (dynamicform_dto.FieldResponse, error) {
	if _, err := s.getOwnedForm(ctx, formID, actorID, perms); err != nil {
		return dynamicform_dto.FieldResponse{}, err
	}
	if !slices.Contains(constants.DynamicFormFieldTypes, req.FieldType) {
		return dynamicform_dto.FieldResponse{}, apperror.BadRequest("Tipe field tidak dikenal")
	}
	values, err := s.fieldValuesFromRequest(req)
	if err != nil {
		return dynamicform_dto.FieldResponse{}, err
	}
	values["isActive"] = 1
	newID, err := s.repo.AddField(ctx, formID, values)
	if err != nil {
		return dynamicform_dto.FieldResponse{}, apperror.Internal("")
	}
	s.bumpVersion(ctx, formID)
	s.logAudit(ctx, actorID, formID, "add_field", nil, values, map[string]int64{"fieldID": newID})
	s.enqueueHeaderSync(ctx, formID)

	field, _ := s.repo.GetField(ctx, formID, newID)
	return toFieldResponse(field), nil
}

func (s *ServiceImpl) UpdateField(ctx context.Context, formID, fieldID int64, req dynamicform_dto.FieldRequest, actorID int64, perms []string) (dynamicform_dto.FieldResponse, error) {
	if _, err := s.getOwnedForm(ctx, formID, actorID, perms); err != nil {
		return dynamicform_dto.FieldResponse{}, err
	}
	existing, err := s.repo.GetField(ctx, formID, fieldID)
	if err != nil {
		return dynamicform_dto.FieldResponse{}, apperror.NotFound("Field tidak ditemukan")
	}

	var values map[string]any
	if existing.IsSystemField {
		// System email field: only label / placeholder / helpText are editable.
		values = map[string]any{
			"label":       strings.TrimSpace(req.Label),
			"placeholder": req.Placeholder,
			"helpText":    req.HelpText,
		}
		if values["label"] == "" {
			values["label"] = sysEmailLabel
		}
	} else {
		if !slices.Contains(constants.DynamicFormFieldTypes, req.FieldType) {
			return dynamicform_dto.FieldResponse{}, apperror.BadRequest("Tipe field tidak dikenal")
		}
		values, err = s.fieldValuesFromRequest(req)
		if err != nil {
			return dynamicform_dto.FieldResponse{}, err
		}
	}
	if err := s.repo.UpdateField(ctx, formID, fieldID, values); err != nil {
		return dynamicform_dto.FieldResponse{}, apperror.Internal("")
	}
	// Replaced image: best-effort delete the old asset.
	if !existing.IsSystemField && existing.FieldType == "image" && req.ImageURL != nil &&
		existing.HelpText != nil && *existing.HelpText != *req.ImageURL {
		_ = s.uploader.DeleteFile(*existing.HelpText)
	}
	s.bumpVersion(ctx, formID)
	s.logAudit(ctx, actorID, formID, "update_field", existing, values, map[string]int64{"fieldID": fieldID})
	if existing.Label != fmt.Sprint(values["label"]) {
		s.enqueueHeaderSync(ctx, formID)
	}

	field, _ := s.repo.GetField(ctx, formID, fieldID)
	return toFieldResponse(field), nil
}

func (s *ServiceImpl) RemoveField(ctx context.Context, formID, fieldID int64, actorID int64, perms []string) error {
	if _, err := s.getOwnedForm(ctx, formID, actorID, perms); err != nil {
		return err
	}
	existing, err := s.repo.GetField(ctx, formID, fieldID)
	if err != nil {
		return apperror.NotFound("Field tidak ditemukan")
	}
	if existing.IsSystemField {
		return apperror.Unprocessable("Field sistem tidak bisa dihapus")
	}
	if err := s.repo.SoftDeleteField(ctx, formID, fieldID); err != nil {
		return apperror.Internal("")
	}
	if existing.FieldType == "image" && existing.HelpText != nil {
		_ = s.uploader.DeleteFile(*existing.HelpText)
	}
	s.bumpVersion(ctx, formID)
	s.logAudit(ctx, actorID, formID, "remove_field", existing, nil, map[string]int64{"fieldID": fieldID})
	s.enqueueHeaderSync(ctx, formID)
	return nil
}

func (s *ServiceImpl) ReorderFields(ctx context.Context, formID int64, order []int64, actorID int64, perms []string) error {
	if _, err := s.getOwnedForm(ctx, formID, actorID, perms); err != nil {
		return err
	}
	if err := s.repo.ReorderFields(ctx, formID, order); err != nil {
		return apperror.Internal("")
	}
	s.bumpVersion(ctx, formID)
	s.logAudit(ctx, actorID, formID, "reorder_fields", nil, map[string]any{"order": order}, nil)
	s.enqueueHeaderSync(ctx, formID)
	return nil
}

func (s *ServiceImpl) bumpVersion(ctx context.Context, formID int64) {
	form, err := s.repo.GetByID(ctx, formID)
	if err != nil {
		return
	}
	_ = s.repo.UpdateForm(ctx, formID, map[string]any{"version": form.Version + 1, "updatedDate": time.Now()})
}
