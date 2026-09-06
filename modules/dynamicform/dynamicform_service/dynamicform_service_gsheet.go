package dynamicform_service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"fsldk-api/base/apperror"
	"fsldk-api/constants"
	"fsldk-api/modules/dynamicform/dynamicform_dto"
	"fsldk-api/modules/dynamicform/dynamicform_model"
	"fsldk-api/modules/dynamicform/dynamicform_repository"
	"fsldk-api/modules/jobqueue/jobqueue_dto"
	"fsldk-api/modules/jobqueue/jobqueue_model"
)

// gsheetLockKey namespaces the per-form advisory lock so one form is never
// processed by two Sheet jobs at once.
func gsheetLockKey(formID int64) string {
	return fmt.Sprintf("dform_gsheet_%d", formID)
}

func tabOf(form dynamicform_model.Form) string {
	if strings.TrimSpace(form.GsheetTabName) == "" {
		return "Responses"
	}
	return form.GsheetTabName
}

// driveNameSanitizer strips characters Drive dislikes in file/folder names and
// keeps the result non-empty.
func driveName(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "/", "-"))
	if s == "" {
		return "untitled"
	}
	if len(s) > 120 {
		s = s[:120]
	}
	return s
}

// ensureSheet builds the full Drive tree for a form (idempotent) — ported from
// ldksyahid-app DynamicFormGDriveService::setupFormFolder:
//
//	<root>/<Form Title>/
//	  ├── <Form Title> — Responses   (spreadsheet)
//	  ├── attachments/  └── <file field label>/
//	  └── assets/       └── <image field label>/
//
// then writes the header row and shares the form folder with the creator and the
// notify-email list. Records gsheetLastSyncError on failure; only gsheet/connect
// surfaces the error to the caller.
func (s *ServiceImpl) ensureSheet(ctx context.Context, form dynamicform_model.Form) error {
	if !s.gsheet.Enabled() {
		return apperror.Unprocessable("Integrasi Google Sheets belum dikonfigurasi di server.")
	}
	haveSheet := form.GsheetSpreadsheetID != nil && *form.GsheetSpreadsheetID != ""
	haveTree := form.GdriveFormFolderID != nil && *form.GdriveFormFolderID != ""
	if haveSheet && haveTree {
		return nil // fully set up
	}
	tab := tabOf(form)
	fail := func(err error) error {
		_ = s.repo.TouchGsheetSync(ctx, form.FormID, err.Error())
		return err
	}

	// 1. form folder inside the configured root folder
	formFolderID, _, err := s.gsheet.CreateFolder(ctx, driveName(form.Title), s.gsheetFolderID)
	if err != nil {
		return fail(err)
	}
	// 2. spreadsheet — reuse an already-created one (pull it into the form
	//    folder), otherwise create it inside the form folder.
	var sheetID, sheetURL string
	if haveSheet {
		sheetID = *form.GsheetSpreadsheetID
		sheetURL = strOr(form.GsheetSpreadsheetURL)
		_ = s.gsheet.MoveFile(ctx, sheetID, formFolderID, s.gsheetFolderID)
	} else {
		sheetID, sheetURL, err = s.gsheet.CreateSpreadsheet(ctx, form.Title+" — Responses", formFolderID)
		if err != nil {
			return fail(err)
		}
	}
	// 3. attachments/ + assets/
	attachID, _, err := s.gsheet.CreateFolder(ctx, "attachments", formFolderID)
	if err != nil {
		return fail(err)
	}
	assetsID, _, err := s.gsheet.CreateFolder(ctx, "assets", formFolderID)
	if err != nil {
		return fail(err)
	}

	if err := s.repo.UpdateForm(ctx, form.FormID, map[string]any{
		"gsheetSpreadsheetID": sheetID, "gsheetSpreadsheetURL": sheetURL, "gsheetTabName": tab,
		"gdriveFormFolderID": formFolderID, "gdriveAttachmentsFolderID": attachID, "gdriveAssetsFolderID": assetsID,
	}); err != nil {
		return err
	}
	form.GsheetSpreadsheetID = &sheetID
	form.GdriveFormFolderID = &formFolderID
	form.GdriveAttachmentsFolderID = &attachID
	form.GdriveAssetsFolderID = &assetsID

	// 4. header row
	fields, _ := s.repo.ListFields(ctx, form.FormID, true)
	if hErr := s.gsheet.SetHeaderRow(ctx, sheetID, tab, buildHeader(fields)); hErr != nil {
		return fail(hErr)
	}
	// 5. per-field subfolders for existing file/image fields
	for _, f := range fields {
		s.ensureFieldFolder(ctx, form, f)
	}
	// 6. share the whole tree (form folder) with creator + notify emails
	if emails := s.sheetShareEmails(ctx, form); len(emails) > 0 {
		_ = s.gsheet.Share(ctx, formFolderID, emails)
	}
	_ = s.repo.TouchGsheetSync(ctx, form.FormID, "")
	return nil
}

// ensureFieldFolder creates the Drive subfolder for one file/image field
// (attachments/<label>/ or assets/<label>/) when the form's tree exists and the
// field has no folder yet. Best-effort — a failure only logs.
func (s *ServiceImpl) ensureFieldFolder(ctx context.Context, form dynamicform_model.Form, field dynamicform_model.Field) {
	if !s.gsheet.Enabled() {
		return
	}
	if field.FieldType != "file" && field.FieldType != "image" {
		return
	}
	if field.GdriveFolderID != nil && *field.GdriveFolderID != "" {
		return
	}
	parent := form.GdriveAttachmentsFolderID
	if field.FieldType == "image" {
		parent = form.GdriveAssetsFolderID
	}
	if parent == nil || *parent == "" {
		return
	}
	subID, _, err := s.gsheet.CreateFolder(ctx, driveName(field.Label), *parent)
	if err != nil {
		log.Printf("[DYNAMICFORM] gdrive subfolder for field %d failed: %v", field.FieldID, err)
		return
	}
	_ = s.repo.UpdateField(ctx, form.FormID, field.FieldID, map[string]any{"gdriveFolderID": subID})
	// NOTE: builder images (fieldType "image") keep their LOCAL helpText URL — a
	// public respondent must be able to render <img src=…>, and Drive files in
	// the restricted form folder are not publicly fetchable. The assets/<label>/
	// subfolder is created for structural parity with the reference only.
}

// relocateSubmissionFiles uploads each locally-stored file of a submission into
// its field's Drive subfolder (attachments/<label>/), then repoints the file row
// + answer to the Drive link and deletes the local copy. No-op when the form has
// no Drive tree. Called post-commit so it never holds a DB lock during upload.
func (s *ServiceImpl) relocateSubmissionFiles(ctx context.Context, form dynamicform_model.Form, submissionID int64, fields []dynamicform_model.Field) {
	if !s.gsheet.Enabled() || form.GdriveAttachmentsFolderID == nil || *form.GdriveAttachmentsFolderID == "" {
		return
	}
	folderByField := map[int64]string{}
	for _, f := range fields {
		if f.GdriveFolderID != nil && *f.GdriveFolderID != "" {
			folderByField[f.FieldID] = *f.GdriveFolderID
		}
	}
	byField, _ := s.repo.FilesFor(ctx, []int64{submissionID})
	for _, fl := range byField[submissionID] {
		if fl.GdriveFileID != nil && *fl.GdriveFileID != "" {
			continue // already on Drive
		}
		folderID, ok := folderByField[fl.FieldID]
		if !ok || folderID == "" {
			continue
		}
		data, err := os.ReadFile(s.uploader.LocalPath(fl.FileURL))
		if err != nil {
			log.Printf("[DYNAMICFORM] read local file %s failed: %v", fl.FileURL, err)
			continue
		}
		mt := ""
		if fl.MimeType != nil {
			mt = *fl.MimeType
		}
		name := fmt.Sprintf("submission_%d_%s", submissionID, driveName(fl.OriginalFileName))
		driveID, driveURL, uErr := s.gsheet.UploadFile(ctx, folderID, name, mt, data)
		if uErr != nil {
			log.Printf("[DYNAMICFORM] upload %s to drive failed: %v", fl.OriginalFileName, uErr)
			continue
		}
		if err := s.repo.RelocateFile(ctx, fl.FileID, submissionID, fl.FieldID, driveURL, driveID); err != nil {
			log.Printf("[DYNAMICFORM] relocate file row %d failed: %v", fl.FileID, err)
			continue
		}
		_ = s.uploader.DeleteFile(fl.FileURL)
	}
}

func (s *ServiceImpl) sheetShareEmails(ctx context.Context, form dynamicform_model.Form) []string {
	seen := map[string]bool{}
	var out []string
	if form.CreatedBy != nil {
		if e := strings.TrimSpace(s.repo.UserEmail(ctx, *form.CreatedBy)); e != "" {
			seen[e] = true
			out = append(out, e)
		}
	}
	for _, e := range notifyEmailsOf(form) {
		e = strings.TrimSpace(e)
		if e != "" && !seen[e] {
			seen[e] = true
			out = append(out, e)
		}
	}
	return out
}

func (s *ServiceImpl) enqueueHeaderSync(ctx context.Context, formID int64) {
	form, err := s.repo.GetByID(ctx, formID)
	if err != nil || !form.GsheetEnabled || form.GsheetSpreadsheetID == nil || !s.gsheet.Enabled() {
		return
	}
	_, _ = s.jobs.Enqueue(ctx, jobqueue_dto.EnqueueInput{
		Queue: jobqueue_model.QueueDefault, JobType: constants.JobDynamicFormGSheetHeader,
		Payload: map[string]int64{"formID": formID},
	})
}

// ---------------------------------------------------------------------------
// CMS endpoints
// ---------------------------------------------------------------------------

func (s *ServiceImpl) gsheetStatus(form dynamicform_model.Form) dynamicform_dto.GSheetStatus {
	return dynamicform_dto.GSheetStatus{
		Enabled: form.GsheetEnabled, SpreadsheetURL: strOr(form.GsheetSpreadsheetURL),
		LastSyncDate: fmtTimePtr(form.GsheetLastSyncDate), LastSyncError: strOr(form.GsheetLastSyncError),
	}
}

func (s *ServiceImpl) GSheetConnect(ctx context.Context, formID int64, actorID int64, perms []string) (dynamicform_dto.GSheetStatus, error) {
	form, err := s.getOwnedForm(ctx, formID, actorID, perms)
	if err != nil {
		return dynamicform_dto.GSheetStatus{}, err
	}
	if !s.gsheet.Enabled() {
		return dynamicform_dto.GSheetStatus{}, apperror.Unprocessable("Integrasi Google Sheets belum dikonfigurasi di server.")
	}
	if err := s.repo.UpdateForm(ctx, formID, map[string]any{"gsheetEnabled": 1}); err != nil {
		return dynamicform_dto.GSheetStatus{}, apperror.Internal("")
	}
	form, _ = s.repo.GetByID(ctx, formID)
	if err := s.ensureSheet(ctx, form); err != nil {
		return dynamicform_dto.GSheetStatus{}, apperror.Unprocessable("Gagal membuat Google Sheet: " + err.Error())
	}
	form, _ = s.repo.GetByID(ctx, formID)
	return s.gsheetStatus(form), nil
}

func (s *ServiceImpl) GSheetResync(ctx context.Context, formID int64, actorID int64, perms []string) (dynamicform_dto.GSheetStatus, error) {
	form, err := s.getOwnedForm(ctx, formID, actorID, perms)
	if err != nil {
		return dynamicform_dto.GSheetStatus{}, err
	}
	if form.GsheetSpreadsheetID == nil {
		return dynamicform_dto.GSheetStatus{}, apperror.Unprocessable("Formulir ini belum terhubung ke Google Sheets.")
	}
	// Build the Drive folder tree if it is missing (forms connected before the
	// folder-tree feature) — ensureSheet is idempotent and pulls the existing
	// spreadsheet into the new form folder.
	if form.GdriveFormFolderID == nil || *form.GdriveFormFolderID == "" {
		_ = s.ensureSheet(ctx, form)
		form, _ = s.repo.GetByID(ctx, formID)
	}
	if _, err := s.jobs.Enqueue(ctx, jobqueue_dto.EnqueueInput{
		Queue: jobqueue_model.QueueDefault, JobType: constants.JobDynamicFormGSheetRebuild,
		Payload: map[string]int64{"formID": formID},
	}); err != nil {
		return dynamicform_dto.GSheetStatus{}, apperror.Internal("")
	}
	return s.gsheetStatus(form), nil
}

func (s *ServiceImpl) GSheetDisconnect(ctx context.Context, formID int64, actorID int64, perms []string) (dynamicform_dto.GSheetStatus, error) {
	form, err := s.getOwnedForm(ctx, formID, actorID, perms)
	if err != nil {
		return dynamicform_dto.GSheetStatus{}, err
	}
	if err := s.repo.UpdateForm(ctx, formID, map[string]any{"gsheetEnabled": 0}); err != nil {
		return dynamicform_dto.GSheetStatus{}, apperror.Internal("")
	}
	form.GsheetEnabled = false
	return s.gsheetStatus(form), nil
}

// ---------------------------------------------------------------------------
// jobqueue handlers (registered on the "default" queue)
// ---------------------------------------------------------------------------

type gsheetPayload struct {
	FormID         int64 `json:"formID"`
	SubmissionID   int64 `json:"submissionID"`
	GsheetRowIndex int64 `json:"gsheetRowIndex"`
}

// runSheetJob loads the form, skips when the mirror is off, and runs fn under
// the per-form advisory lock. It clears/sets gsheetLastSyncError accordingly.
func (s *ServiceImpl) runSheetJob(ctx context.Context, formID int64, fn func(form dynamicform_model.Form, spreadsheetID, tab string) error) error {
	if !s.gsheet.Enabled() {
		return nil
	}
	form, err := s.repo.GetByID(ctx, formID)
	if err != nil {
		return nil // form gone — nothing to mirror
	}
	if !form.GsheetEnabled || form.GsheetSpreadsheetID == nil || *form.GsheetSpreadsheetID == "" {
		return nil
	}
	lockErr := s.repo.WithAdvisoryLock(ctx, gsheetLockKey(formID), 0, func() error {
		return fn(form, *form.GsheetSpreadsheetID, tabOf(form))
	})
	if lockErr == dynamicform_repository.ErrLockBusy {
		return fmt.Errorf("gsheet: form %d busy, will retry", formID) // retryable
	}
	if lockErr != nil {
		_ = s.repo.TouchGsheetSync(ctx, formID, lockErr.Error())
		return lockErr
	}
	_ = s.repo.TouchGsheetSync(ctx, formID, "")
	return nil
}

func (s *ServiceImpl) sheetRowFor(ctx context.Context, submissionID int64) (dynamicform_model.Submission, []string, error) {
	// The submission's formID is not on the payload for append/update, so fetch
	// it via a form-agnostic lookup path: list answers, then resolve the form.
	var sub dynamicform_model.Submission
	if err := s.repo.DB().WithContext(ctx).Table(constants.TableDynamicFormSubmission).
		Where("submissionID = ?", submissionID).Take(&sub).Error; err != nil {
		return sub, nil, err
	}
	fields, _ := s.repo.ListFields(ctx, sub.FormID, true)
	answers, _ := s.repo.AnswersFor(ctx, []int64{submissionID})
	return sub, buildRow(sub, answers[submissionID], fields), nil
}

func (s *ServiceImpl) HandleGSheetAppendJob(ctx context.Context, payload string) error {
	var p gsheetPayload
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return err
	}
	sub, row, err := s.sheetRowFor(ctx, p.SubmissionID)
	if err != nil {
		return nil // submission gone
	}
	return s.runSheetJob(ctx, sub.FormID, func(_ dynamicform_model.Form, id, tab string) error {
		rowIndex, aErr := s.gsheet.AppendRow(ctx, id, tab, row)
		if aErr != nil {
			return aErr
		}
		if rowIndex > 0 {
			_ = s.repo.SetGsheetRowIndex(ctx, p.SubmissionID, rowIndex)
		}
		return nil
	})
}

func (s *ServiceImpl) HandleGSheetUpdateJob(ctx context.Context, payload string) error {
	var p gsheetPayload
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return err
	}
	sub, row, err := s.sheetRowFor(ctx, p.SubmissionID)
	if err != nil {
		return nil
	}
	return s.runSheetJob(ctx, sub.FormID, func(_ dynamicform_model.Form, id, tab string) error {
		rowIndex := 0
		if sub.GsheetRowIndex != nil {
			rowIndex = *sub.GsheetRowIndex
		}
		if rowIndex == 0 {
			rowIndex, _ = s.gsheet.FindRowBySubmissionID(ctx, id, tab, p.SubmissionID)
		}
		if rowIndex == 0 {
			newIndex, aErr := s.gsheet.AppendRow(ctx, id, tab, row)
			if aErr != nil {
				return aErr
			}
			if newIndex > 0 {
				_ = s.repo.SetGsheetRowIndex(ctx, p.SubmissionID, newIndex)
			}
			return nil
		}
		if uErr := s.gsheet.UpdateRowByIndex(ctx, id, tab, rowIndex, row); uErr != nil {
			return uErr
		}
		_ = s.repo.SetGsheetRowIndex(ctx, p.SubmissionID, rowIndex)
		return nil
	})
}

func (s *ServiceImpl) HandleGSheetDeleteJob(ctx context.Context, payload string) error {
	var p gsheetPayload
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return err
	}
	return s.runSheetJob(ctx, p.FormID, func(_ dynamicform_model.Form, id, tab string) error {
		rowIndex, fErr := s.gsheet.FindRowBySubmissionID(ctx, id, tab, p.SubmissionID)
		if fErr != nil {
			return fErr
		}
		if rowIndex == 0 {
			return nil // already gone
		}
		if dErr := s.gsheet.DeleteRowByIndex(ctx, id, tab, rowIndex); dErr != nil {
			return dErr
		}
		return s.repo.DecrementRowIndexesAfter(ctx, p.FormID, rowIndex)
	})
}

func (s *ServiceImpl) HandleGSheetHeaderJob(ctx context.Context, payload string) error {
	var p gsheetPayload
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return err
	}
	return s.runSheetJob(ctx, p.FormID, func(_ dynamicform_model.Form, id, tab string) error {
		fields, _ := s.repo.ListFields(ctx, p.FormID, true)
		return s.gsheet.ReorderColumns(ctx, id, tab, buildHeader(fields))
	})
}

func (s *ServiceImpl) HandleGSheetRebuildJob(ctx context.Context, payload string) error {
	var p gsheetPayload
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return err
	}
	return s.runSheetJob(ctx, p.FormID, func(_ dynamicform_model.Form, id, tab string) error {
		fields, _ := s.repo.ListFields(ctx, p.FormID, true)
		if hErr := s.gsheet.SetHeaderRow(ctx, id, tab, buildHeader(fields)); hErr != nil {
			return hErr
		}
		if cErr := s.gsheet.ClearDataRows(ctx, id, tab); cErr != nil {
			return cErr
		}
		subs, _ := s.repo.AllSubmissionsAsc(ctx, p.FormID)
		ids := make([]int64, len(subs))
		for i, sub := range subs {
			ids[i] = sub.SubmissionID
		}
		answers, _ := s.repo.AnswersFor(ctx, ids)
		for _, sub := range subs {
			rowIndex, aErr := s.gsheet.AppendRow(ctx, id, tab, buildRow(sub, answers[sub.SubmissionID], fields))
			if aErr != nil {
				return aErr
			}
			if rowIndex > 0 {
				_ = s.repo.SetGsheetRowIndex(ctx, sub.SubmissionID, rowIndex)
			}
		}
		log.Printf("[DYNAMICFORM] gsheet rebuild done for form %d (%d rows)", p.FormID, len(subs))
		return nil
	})
}
