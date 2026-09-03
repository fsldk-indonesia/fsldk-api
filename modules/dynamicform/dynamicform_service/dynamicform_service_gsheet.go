package dynamicform_service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

// ensureSheet creates the spreadsheet for a form (idempotent) and writes its
// header + shares it. It records gsheetLastSyncError on failure but only the
// gsheet/connect endpoint surfaces that error to the caller.
func (s *ServiceImpl) ensureSheet(ctx context.Context, form dynamicform_model.Form) error {
	if !s.gsheet.Enabled() {
		return apperror.Unprocessable("Integrasi Google Sheets belum dikonfigurasi di server.")
	}
	if form.GsheetSpreadsheetID != nil && *form.GsheetSpreadsheetID != "" {
		return nil
	}
	tab := tabOf(form)
	id, url, err := s.gsheet.CreateSpreadsheet(ctx, form.Title+" — Responses", s.gsheetFolderID)
	if err != nil {
		_ = s.repo.TouchGsheetSync(ctx, form.FormID, err.Error())
		return err
	}
	if err := s.repo.UpdateForm(ctx, form.FormID, map[string]any{
		"gsheetSpreadsheetID": id, "gsheetSpreadsheetURL": url, "gsheetTabName": tab,
	}); err != nil {
		return err
	}
	fields, _ := s.repo.ListFields(ctx, form.FormID, true)
	if hErr := s.gsheet.SetHeaderRow(ctx, id, tab, buildHeader(fields)); hErr != nil {
		_ = s.repo.TouchGsheetSync(ctx, form.FormID, hErr.Error())
		return hErr
	}
	if emails := s.sheetShareEmails(ctx, form); len(emails) > 0 {
		_ = s.gsheet.Share(ctx, id, emails)
	}
	_ = s.repo.TouchGsheetSync(ctx, form.FormID, "")
	return nil
}

func (s *ServiceImpl) sheetShareEmails(ctx context.Context, form dynamicform_model.Form) []string {
	seen := map[string]bool{}
	var out []string
	for _, e := range notifyEmailsOf(form) {
		e = strings.TrimSpace(e)
		if e != "" && !seen[e] {
			seen[e] = true
			out = append(out, e)
		}
	}
	collabs, _ := s.repo.ListCollaborators(ctx, form.FormID)
	for _, c := range collabs {
		if c.UserEmail != "" && !seen[c.UserEmail] {
			seen[c.UserEmail] = true
			out = append(out, c.UserEmail)
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
