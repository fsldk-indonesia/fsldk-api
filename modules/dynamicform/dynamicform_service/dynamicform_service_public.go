package dynamicform_service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"net/mail"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fsldk-api/base/apperror"
	"fsldk-api/constants"
	"fsldk-api/modules/dynamicform/dynamicform_dto"
	"fsldk-api/modules/dynamicform/dynamicform_model"
	"fsldk-api/modules/dynamicform/dynamicform_repository"
	"fsldk-api/modules/jobqueue/jobqueue_dto"
	"fsldk-api/modules/jobqueue/jobqueue_model"
	"fsldk-api/pkg/mailer"
)

// CodeFormClosed marks a 422 raised because the form is not accepting responses
// (frontend renders a "Formulir Ditutup" card instead of an inline banner).
const CodeFormClosed = "42-CLOSED"

var imageExt = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true}

// ---------------------------------------------------------------------------
// public: render
// ---------------------------------------------------------------------------

func (s *ServiceImpl) isAcceptingSubmissions(form dynamicform_model.Form) bool {
	now := time.Now()
	if form.Status != constants.DynamicFormStatusPublished || !form.IsActive {
		return false
	}
	if form.StartDate != nil && now.Before(*form.StartDate) {
		return false
	}
	if form.EndDate != nil && now.After(*form.EndDate) {
		return false
	}
	if form.MaxSubmission != nil && form.TotalSubmission >= *form.MaxSubmission {
		return false
	}
	return true
}

func (s *ServiceImpl) isPrivileged(ctx context.Context, form dynamicform_model.Form, userID *int64) bool {
	if userID == nil {
		return false
	}
	if form.CreatedBy != nil && *form.CreatedBy == *userID {
		return true
	}
	ok, _ := s.repo.IsCollaborator(ctx, form.FormID, *userID)
	return ok
}

func (s *ServiceImpl) GetPublicForm(ctx context.Context, slug string, authUserID *int64, authUserEmail *string) (dynamicform_dto.PublicFormResponse, error) {
	form, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return dynamicform_dto.PublicFormResponse{}, apperror.NotFound("Formulir tidak ditemukan")
	}
	privileged := s.isPrivileged(ctx, form, authUserID)
	accepting := s.isAcceptingSubmissions(form)

	if !privileged {
		if form.RequireLogin && authUserID == nil {
			return dynamicform_dto.PublicFormResponse{}, apperror.Unauthorized("Masuk untuk mengisi formulir ini")
		}
		if form.Status != constants.DynamicFormStatusPublished {
			return dynamicform_dto.PublicFormResponse{}, apperror.NotFound("Formulir tidak ditemukan atau sudah ditutup")
		}
		if !accepting {
			e := apperror.Unprocessable("Formulir ini sedang tidak menerima tanggapan.")
			e.Code = CodeFormClosed
			return dynamicform_dto.PublicFormResponse{}, e
		}
	}

	fields, _ := s.repo.ListFields(ctx, form.FormID, true)
	sections, _ := s.repo.ListSections(ctx, form.FormID)

	resp := dynamicform_dto.PublicFormResponse{
		FormID: form.FormID, Title: form.Title, Description: strOr(form.Description),
		Slug: form.Slug, Status: form.Status, RequireLogin: form.RequireLogin,
		IsMultipleSubmit: form.IsMultipleSubmit, Version: form.Version, IsPreview: privileged && !accepting,
	}
	for _, sec := range sections {
		resp.Sections = append(resp.Sections, dynamicform_dto.PublicSection{
			SectionID: sec.SectionID, Title: sec.Title, Description: sec.Description, SortOrder: sec.SortOrder,
		})
	}
	for _, f := range fields {
		resp.Fields = append(resp.Fields, dynamicform_dto.PublicField{
			FieldID: f.FieldID, SectionID: f.SectionID, FieldType: f.FieldType, Label: f.Label,
			Placeholder: f.Placeholder, HelpText: f.HelpText, IsRequired: f.IsRequired,
			IsSystemField: f.IsSystemField, SortOrder: f.SortOrder,
			Options: rawJSON(f.OptionsJSON), Validation: rawJSON(f.ValidationJSON),
			DefaultValue: f.DefaultValue, ConditionalLogic: rawJSON(f.ConditionalLogicJSON),
			FieldConfig: rawJSON(f.FieldConfigJSON),
		})
	}
	if authUserID != nil && authUserEmail != nil {
		resp.PrefillEmail = *authUserEmail
	}
	if authUserID != nil {
		resp.DraftAnswers = s.loadDraftAnswers(ctx, form.FormID, *authUserID)
	}
	return resp, nil
}

// loadDraftAnswers reads the caller's draft, drops staged file entries whose
// file has been swept from disk, persists the shrink, and returns a
// frontend-friendly map (file entries -> {fileURL, originalFileName}).
func (s *ServiceImpl) loadDraftAnswers(ctx context.Context, formID, userID int64) map[string]json.RawMessage {
	draft, ok, err := s.repo.GetDraft(ctx, formID, userID)
	if err != nil || !ok {
		return nil
	}
	current := map[string]json.RawMessage{}
	if json.Unmarshal([]byte(draft.AnswersJSON), &current) != nil {
		return nil
	}
	out := map[string]json.RawMessage{}
	changed := false
	for k, v := range current {
		if url, name, _, _, isFile := stagedFileEntry(v); isFile {
			if _, statErr := s.uploader.FileSize(url); statErr != nil {
				changed = true
				continue
			}
			b, _ := json.Marshal(map[string]any{"fileURL": url, "originalFileName": name})
			out[k] = b
			continue
		}
		out[k] = v
	}
	if changed {
		_ = s.repo.MutateDraft(ctx, formID, userID, func(cur map[string]any) (map[string]any, error) {
			for k := range cur {
				if _, ok := out[k]; !ok {
					delete(cur, k)
				}
			}
			return cur, nil
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// stagedFileEntry inspects a decoded draft value and reports whether it is a
// staged-file marker.
func stagedFileEntry(v any) (fileURL, name, mimeType string, sizeKB int, ok bool) {
	var raw []byte
	switch t := v.(type) {
	case json.RawMessage:
		raw = t
	case []byte:
		raw = t
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return "", "", "", 0, false
		}
		raw = b
	}
	var entry struct {
		File     bool   `json:"__file"`
		FileURL  string `json:"fileURL"`
		Name     string `json:"originalFileName"`
		MimeType string `json:"mimeType"`
		SizeKB   int    `json:"fileSizeKB"`
	}
	if json.Unmarshal(raw, &entry) != nil || !entry.File {
		return "", "", "", 0, false
	}
	return entry.FileURL, entry.Name, entry.MimeType, entry.SizeKB, true
}

// ---------------------------------------------------------------------------
// public: submit
// ---------------------------------------------------------------------------

func (s *ServiceImpl) Submit(ctx context.Context, slug string, in SubmitInput) (dynamicform_dto.SubmitResult, error) {
	form, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return dynamicform_dto.SubmitResult{}, apperror.NotFound("Formulir tidak ditemukan")
	}
	success := dynamicform_dto.SubmitResult{
		Slug: form.Slug, RedirectURL: strOr(form.RedirectURL),
		ConfirmationMessage: strOr(form.ConfirmationMessage), IsMultipleSubmit: form.IsMultipleSubmit,
	}

	// Anti-spam — honeypot & timing return a silent fake success.
	if strings.TrimSpace(in.Honeypot) != "" {
		log.Printf("[DYNAMICFORM] honeypot tripped on form %d (ip=%s)", form.FormID, in.IP)
		return success, nil
	}
	if in.FormTS > 0 {
		if elapsed := time.Now().UnixMilli() - in.FormTS; elapsed >= 0 && elapsed < 3000 {
			log.Printf("[DYNAMICFORM] submit too fast (%dms) on form %d (ip=%s)", elapsed, form.FormID, in.IP)
			return success, nil
		}
	}

	privileged := s.isPrivileged(ctx, form, in.AuthUserID)
	if form.RequireLogin && in.AuthUserID == nil {
		return dynamicform_dto.SubmitResult{}, apperror.Unauthorized("Masuk untuk mengisi formulir ini")
	}
	if !privileged && !s.isAcceptingSubmissions(form) {
		e := apperror.Unprocessable("Formulir ini sudah ditutup.")
		e.Code = CodeFormClosed
		return dynamicform_dto.SubmitResult{}, e
	}

	fields, _ := s.repo.ListFields(ctx, form.FormID, true)
	sysField, hasSys := systemEmailField(fields)

	// Rate-limit per IP (skipped for privileged callers doing QA).
	if !privileged {
		window := time.Duration(form.RateLimitWindowMinutes) * time.Minute
		since := time.Now().Add(-window)
		count, _ := s.repo.CountSubmissionsSince(ctx, form.FormID, in.IP, since)
		if count >= form.RateLimitPerIP {
			retry := 60
			if oldest, ok, _ := s.repo.OldestSubmissionSince(ctx, form.FormID, in.IP, since); ok {
				if d := int(time.Until(oldest.Add(window)).Seconds()); d > 0 {
					retry = d
				}
			}
			e := apperror.TooManyRequests("Terlalu banyak pengiriman formulir. Coba lagi beberapa saat lagi.")
			e.Fields = []apperror.FieldError{{Attribute: "retryAfterSeconds", Code: constants.CodeTooManyRequest, Message: strconv.Itoa(retry)}}
			return dynamicform_dto.SubmitResult{}, e
		}
	}

	// Resolve respondent email.
	var email string
	if form.RequireLogin && in.AuthUserEmail != nil {
		email = strings.TrimSpace(*in.AuthUserEmail)
	} else if hasSys {
		email = firstVal(in.Values, sysField.FieldID)
	}
	if _, addrErr := mail.ParseAddress(email); addrErr != nil {
		return dynamicform_dto.SubmitResult{}, apperror.Validation("Data tidak valid", []apperror.FieldError{{
			Attribute: sysFieldAttr(sysField, hasSys), Code: constants.CodeValidationError,
			Message: "Alamat email wajib diisi dengan format yang benar",
		}})
	}

	// Single-submission dedup.
	var dedupUserID *int64
	if form.RequireLogin {
		dedupUserID = in.AuthUserID
	}
	if !form.IsMultipleSubmit && !privileged {
		if dup, _ := s.repo.HasSubmitted(ctx, form.FormID, email, dedupUserID); dup {
			e := apperror.New(http.StatusConflict, constants.CodeDuplicateSubmission,
				fmt.Sprintf("Jazakallah khair, email %s sudah pernah mengisi formulir ini.", email))
			e.Fields = []apperror.FieldError{{Attribute: "alreadySubmitted", Code: constants.CodeDuplicateSubmission, Message: "1"}}
			return dynamicform_dto.SubmitResult{}, e
		}
	}

	// Staged draft files for login users.
	var draftFiles map[int64]stagedFile
	if in.AuthUserID != nil {
		draftFiles = s.stagedFilesOf(ctx, form.FormID, *in.AuthUserID)
	}

	// File-satisfaction map for validation.
	fileSatisfied := map[int64]bool{}
	for _, f := range fields {
		if f.FieldType != "file" {
			continue
		}
		if in.Files[f.FieldID] != nil {
			fileSatisfied[f.FieldID] = true
		} else if sf, ok := draftFiles[f.FieldID]; ok {
			if _, statErr := s.uploader.FileSize(sf.FileURL); statErr == nil {
				fileSatisfied[f.FieldID] = true
			}
		}
	}

	if errs := validateAnswers(fields, in.Values, fileSatisfied); len(errs) > 0 {
		return dynamicform_dto.SubmitResult{}, apperror.Validation("Data tidak valid", errs)
	}

	// Build answers + files.
	answers := map[int64]string{}
	files := map[int64]dynamicform_model.File{}
	for _, f := range fields {
		if isDisplayType(f.FieldType) {
			continue
		}
		if f.FieldType == "file" {
			var fileRow *dynamicform_model.File
			if fh := in.Files[f.FieldID]; fh != nil {
				url, saveErr := s.saveFieldFile(fh)
				if saveErr != nil {
					return dynamicform_dto.SubmitResult{}, apperror.Validation("Data tidak valid", []apperror.FieldError{{
						Attribute: "field_" + strconv.FormatInt(f.FieldID, 10), Code: constants.CodeValidationError,
						Message: saveErr.Error(),
					}})
				}
				sizeKB := int(fh.Size / 1024)
				mt := mimeOf(fh.Filename)
				fileRow = &dynamicform_model.File{FileURL: url, OriginalFileName: fh.Filename, MimeType: &mt, FileSizeKB: &sizeKB}
			} else if sf, ok := draftFiles[f.FieldID]; ok {
				if _, statErr := s.uploader.FileSize(sf.FileURL); statErr == nil {
					mt, size := sf.MimeType, sf.SizeKB
					fileRow = &dynamicform_model.File{FileURL: sf.FileURL, OriginalFileName: sf.Name, MimeType: &mt, FileSizeKB: &size}
				}
			}
			if fileRow != nil {
				files[f.FieldID] = *fileRow
			}
			continue
		}
		if f.FieldType == "checkbox" {
			vals := allVals(in.Values, f.FieldID)
			if len(vals) == 0 {
				continue
			}
			b, _ := json.Marshal(vals)
			answers[f.FieldID] = string(b)
			continue
		}
		if v := firstVal(in.Values, f.FieldID); v != "" {
			answers[f.FieldID] = v
		}
	}

	respondentName := guessRespondentName(fields, in.Values)

	// Idempotency guard: serialise submits per (form, user/IP).
	lockKey := fmt.Sprintf("dform_submit_%d_", form.FormID)
	if dedupUserID != nil {
		lockKey += "u" + strconv.FormatInt(*dedupUserID, 10)
	} else {
		lockKey += "ip" + in.IP
	}
	if len(lockKey) > 64 {
		lockKey = lockKey[:64]
	}

	var submissionID int64
	lockErr := s.repo.WithAdvisoryLock(ctx, lockKey, 0, func() error {
		if !form.IsMultipleSubmit && !privileged {
			if dup, _ := s.repo.HasSubmitted(ctx, form.FormID, email, dedupUserID); dup {
				return apperror.New(http.StatusConflict, constants.CodeDuplicateSubmission,
					fmt.Sprintf("Jazakallah khair, email %s sudah pernah mengisi formulir ini.", email))
			}
		}
		var delDraft *int64
		if in.AuthUserID != nil {
			delDraft = in.AuthUserID
		}
		id, sErr := s.repo.Submit(ctx, dynamicform_repository.SubmitData{
			FormID: form.FormID, FormVersion: form.Version, RespondentEmail: email,
			RespondentName: respondentName, RespondentUserID: dedupUserID,
			IPAddress: in.IP, UserAgent: in.UserAgent, IsValid: true,
			Answers: answers, Files: files, DeleteDraftUserID: delDraft,
		})
		if sErr != nil {
			return sErr
		}
		submissionID = id
		return nil
	})
	if lockErr != nil {
		if lockErr == dynamicform_repository.ErrLockBusy {
			return dynamicform_dto.SubmitResult{}, apperror.TooManyRequests("Formulir Anda sedang diproses, mohon tunggu.")
		}
		if appErr, ok := lockErr.(*apperror.AppError); ok {
			return dynamicform_dto.SubmitResult{}, appErr
		}
		return dynamicform_dto.SubmitResult{}, apperror.Internal("Gagal menyimpan tanggapan")
	}

	s.afterSubmit(ctx, form, submissionID, email, strDeref(respondentName), fields, answers, files)
	return success, nil
}

func (s *ServiceImpl) afterSubmit(ctx context.Context, form dynamicform_model.Form, submissionID int64, email, name string, fields []dynamicform_model.Field, answers map[int64]string, files map[int64]dynamicform_model.File) {
	if form.SendConfirmationEmail && s.mailer != nil {
		var pairs []mailer.AnswerPair
		for _, f := range fields {
			if isDisplayType(f.FieldType) || f.FieldType == "file" {
				if f.FieldType == "file" {
					if fr, ok := files[f.FieldID]; ok {
						pairs = append(pairs, mailer.AnswerPair{Label: f.Label, Value: fr.OriginalFileName})
					}
				}
				continue
			}
			if v, ok := answers[f.FieldID]; ok {
				pairs = append(pairs, mailer.AnswerPair{Label: f.Label, Value: displayValue(v)})
			}
		}
		if err := s.mailer.SendFormSubmissionConfirmation(email, name, form.Title, pairs, time.Now().Format(timeLayout)); err != nil {
			log.Printf("[DYNAMICFORM] confirmation email to %s failed: %v", email, err)
		}
	}
	if form.GsheetEnabled && form.GsheetSpreadsheetID != nil && s.gsheet.Enabled() {
		if _, err := s.jobs.Enqueue(ctx, jobqueue_dto.EnqueueInput{
			Queue: jobqueue_model.QueueDefault, JobType: constants.JobDynamicFormGSheetAppend,
			Payload: map[string]int64{"submissionID": submissionID},
		}); err != nil {
			log.Printf("[DYNAMICFORM] enqueue gsheet append failed: %v", err)
		}
	}
}

// ---------------------------------------------------------------------------
// public: draft autosave
// ---------------------------------------------------------------------------

func (s *ServiceImpl) SaveDraft(ctx context.Context, slug string, userID int64, answers map[string]json.RawMessage) error {
	form, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return apperror.NotFound("Formulir tidak ditemukan")
	}
	fields, _ := s.repo.ListFields(ctx, form.FormID, true)
	allowed := map[string]bool{}
	for _, f := range fields {
		if isDisplayType(f.FieldType) || f.FieldType == "file" || f.IsSystemField {
			continue
		}
		allowed["field_"+strconv.FormatInt(f.FieldID, 10)] = true
	}
	return s.repo.MutateDraft(ctx, form.FormID, userID, func(current map[string]any) (map[string]any, error) {
		for k, raw := range answers {
			if !allowed[k] {
				continue
			}
			if isEmptyRaw(raw) {
				delete(current, k)
				continue
			}
			var v any
			if json.Unmarshal(raw, &v) != nil {
				continue
			}
			current[k] = v
		}
		return current, nil
	})
}

func (s *ServiceImpl) StageDraftFile(ctx context.Context, slug string, userID, fieldID int64, fh *multipart.FileHeader) (string, error) {
	form, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return "", apperror.NotFound("Formulir tidak ditemukan")
	}
	field, err := s.repo.GetField(ctx, form.FormID, fieldID)
	if err != nil || !field.IsActive || field.FieldType != "file" {
		return "", apperror.Unprocessable("Field unggahan tidak ditemukan")
	}
	if err := validateUploadAgainst(fh, field.ValidationJSON); err != nil {
		return "", apperror.Unprocessable(err.Error())
	}
	key := "field_" + strconv.FormatInt(fieldID, 10)
	mutErr := s.repo.MutateDraft(ctx, form.FormID, userID, func(current map[string]any) (map[string]any, error) {
		if oldURL, _, _, _, ok := stagedFileEntry(current[key]); ok && oldURL != "" {
			_ = s.uploader.DeleteFile(oldURL)
		}
		url, saveErr := s.saveFieldFile(fh)
		if saveErr != nil {
			return nil, saveErr
		}
		current[key] = map[string]any{
			"__file": true, "fileURL": url, "originalFileName": fh.Filename,
			"mimeType": mimeOf(fh.Filename), "fileSizeKB": int(fh.Size / 1024),
		}
		return current, nil
	})
	if mutErr != nil {
		return "", apperror.Unprocessable(mutErr.Error())
	}
	return fh.Filename, nil
}

func (s *ServiceImpl) RemoveDraftFile(ctx context.Context, slug string, userID, fieldID int64) error {
	form, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return apperror.NotFound("Formulir tidak ditemukan")
	}
	key := "field_" + strconv.FormatInt(fieldID, 10)
	return s.repo.MutateDraft(ctx, form.FormID, userID, func(current map[string]any) (map[string]any, error) {
		if url, _, _, _, ok := stagedFileEntry(current[key]); ok && url != "" {
			_ = s.uploader.DeleteFile(url)
		}
		delete(current, key)
		return current, nil
	})
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

type stagedFile struct {
	FileURL  string
	Name     string
	MimeType string
	SizeKB   int
}

func (s *ServiceImpl) stagedFilesOf(ctx context.Context, formID, userID int64) map[int64]stagedFile {
	out := map[int64]stagedFile{}
	draft, ok, err := s.repo.GetDraft(ctx, formID, userID)
	if err != nil || !ok {
		return out
	}
	m := map[string]json.RawMessage{}
	if json.Unmarshal([]byte(draft.AnswersJSON), &m) != nil {
		return out
	}
	for k, v := range m {
		if !strings.HasPrefix(k, "field_") {
			continue
		}
		id, convErr := strconv.ParseInt(strings.TrimPrefix(k, "field_"), 10, 64)
		if convErr != nil {
			continue
		}
		if url, name, mt, size, isFile := stagedFileEntry(v); isFile {
			out[id] = stagedFile{FileURL: url, Name: name, MimeType: mt, SizeKB: size}
		}
	}
	return out
}

func (s *ServiceImpl) saveFieldFile(fh *multipart.FileHeader) (string, error) {
	if imageExt[strings.ToLower(filepath.Ext(fh.Filename))] {
		return s.uploader.SaveImage(fh)
	}
	return s.uploader.SaveDocument(fh)
}

func mimeOf(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".pdf":
		return "application/pdf"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	default:
		return "application/octet-stream"
	}
}

func validateUploadAgainst(fh *multipart.FileHeader, validationJSON *string) error {
	if validationJSON == nil || strings.TrimSpace(*validationJSON) == "" {
		return nil
	}
	var v fieldValidation
	if json.Unmarshal([]byte(*validationJSON), &v) != nil {
		return nil
	}
	if len(v.AcceptedTypes) > 0 {
		ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(fh.Filename)), ".")
		ok := false
		for _, t := range v.AcceptedTypes {
			if strings.TrimPrefix(strings.ToLower(strings.TrimSpace(t)), ".") == ext {
				ok = true
				break
			}
		}
		if !ok {
			return fmt.Errorf("format berkas tidak diizinkan (hanya %s)", strings.Join(v.AcceptedTypes, ", "))
		}
	}
	if v.MaxSizeKB != nil && *v.MaxSizeKB > 0 && fh.Size > int64(*v.MaxSizeKB)*1024 {
		return fmt.Errorf("ukuran berkas melebihi %d KB", *v.MaxSizeKB)
	}
	return nil
}

func systemEmailField(fields []dynamicform_model.Field) (dynamicform_model.Field, bool) {
	for _, f := range fields {
		if f.IsSystemField {
			return f, true
		}
	}
	return dynamicform_model.Field{}, false
}

func sysFieldAttr(f dynamicform_model.Field, has bool) string {
	if has {
		return "field_" + strconv.FormatInt(f.FieldID, 10)
	}
	return "email"
}

var nameLabels = map[string]bool{"nama": true, "name": true, "nama lengkap": true, "full name": true, "fullname": true}

func guessRespondentName(fields []dynamicform_model.Field, values map[int64][]string) *string {
	for _, f := range fields {
		if nameLabels[strings.ToLower(strings.TrimSpace(f.Label))] {
			if v := firstVal(values, f.FieldID); v != "" {
				return &v
			}
		}
	}
	return nil
}

func isEmptyRaw(raw json.RawMessage) bool {
	s := strings.TrimSpace(string(raw))
	return s == "" || s == "null" || s == `""` || s == "[]" || s == "{}"
}

func strDeref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// displayValue renders a stored answer for humans (a JSON array becomes "a, b").
func displayValue(v string) string {
	trimmed := strings.TrimSpace(v)
	if strings.HasPrefix(trimmed, "[") {
		var arr []string
		if json.Unmarshal([]byte(trimmed), &arr) == nil {
			return strings.Join(arr, ", ")
		}
	}
	return v
}
