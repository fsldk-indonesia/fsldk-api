// Package dynamicform_service holds the dynamicform business logic.
package dynamicform_service

import (
	"context"
	"encoding/json"
	"io"
	"mime/multipart"

	"fsldk-api/base/dto"
	"fsldk-api/modules/dynamicform/dynamicform_dto"
)

// SubmitInput is the fully-parsed public submit request.
type SubmitInput struct {
	Values        map[int64][]string
	Files         map[int64]*multipart.FileHeader
	IP            string
	UserAgent     string
	Honeypot      string
	FormTS        int64
	AuthUserID    *int64
	AuthUserEmail *string
}

// Service is the dynamicform business-logic contract.
type Service interface {
	// --- CMS: forms ---
	CreateForm(ctx context.Context, req dynamicform_dto.FormRequest, actorID int64, perms []string) (dynamicform_dto.FormResponse, error)
	UpdateForm(ctx context.Context, id int64, req dynamicform_dto.FormRequest, actorID int64, perms []string) (dynamicform_dto.FormResponse, error)
	SetStatus(ctx context.Context, id int64, status string, actorID int64, perms []string) error
	DeleteForm(ctx context.Context, id int64, actorID int64, perms []string) error
	BulkDelete(ctx context.Context, ids []int64, actorID int64, perms []string) (dynamicform_dto.BulkDeleteResult, error)
	ListForms(ctx context.Context, q dto.ListQuery, f dynamicform_dto.FormFilter, actorID int64, perms []string) ([]dynamicform_dto.FormResponse, int, error)
	GetForm(ctx context.Context, id int64, actorID int64, perms []string) (dynamicform_dto.FormResponse, error)

	// --- Builder: fields ---
	AddField(ctx context.Context, formID int64, req dynamicform_dto.FieldRequest, actorID int64, perms []string) (dynamicform_dto.FieldResponse, error)
	UpdateField(ctx context.Context, formID, fieldID int64, req dynamicform_dto.FieldRequest, actorID int64, perms []string) (dynamicform_dto.FieldResponse, error)
	RemoveField(ctx context.Context, formID, fieldID int64, actorID int64, perms []string) error
	ReorderFields(ctx context.Context, formID int64, order []int64, actorID int64, perms []string) error

	// --- Public ---
	GetPublicForm(ctx context.Context, slug string, authUserID *int64, authUserEmail *string) (dynamicform_dto.PublicFormResponse, error)
	Submit(ctx context.Context, slug string, in SubmitInput) (dynamicform_dto.SubmitResult, error)
	SaveDraft(ctx context.Context, slug string, userID int64, answers map[string]json.RawMessage) error
	StageDraftFile(ctx context.Context, slug string, userID, fieldID int64, fh *multipart.FileHeader) (string, error)
	RemoveDraftFile(ctx context.Context, slug string, userID, fieldID int64) error

	// --- Rekap / analytics / export ---
	ListSubmissions(ctx context.Context, formID int64, q dto.ListQuery, f dynamicform_dto.SubmissionFilter, actorID int64, perms []string) ([]dynamicform_dto.SubmissionRow, int, error)
	GetSubmission(ctx context.Context, formID, submissionID int64, actorID int64, perms []string) (dynamicform_dto.SubmissionDetail, error)
	UpdateSubmission(ctx context.Context, formID, submissionID int64, actorID int64, perms []string, req dynamicform_dto.EditSubmissionRequest, files map[int64]*multipart.FileHeader) error
	DeleteSubmission(ctx context.Context, formID, submissionID int64, actorID int64, perms []string) error
	ExportCSV(ctx context.Context, formID int64, actorID int64, perms []string, w io.Writer) (string, error)
	DeleteResponses(ctx context.Context, formID int64, actorID int64, perms []string) error
	GetAnalytics(ctx context.Context, formID int64, actorID int64, perms []string) (dynamicform_dto.Analytics, error)

	// --- Google Sheets ---
	GSheetConnect(ctx context.Context, formID int64, actorID int64, perms []string) (dynamicform_dto.GSheetStatus, error)
	GSheetResync(ctx context.Context, formID int64, actorID int64, perms []string) (dynamicform_dto.GSheetStatus, error)
	GSheetDisconnect(ctx context.Context, formID int64, actorID int64, perms []string) (dynamicform_dto.GSheetStatus, error)

	// --- jobqueue handlers (registered on the "default" queue) ---
	HandleGSheetAppendJob(ctx context.Context, payload string) error
	HandleGSheetUpdateJob(ctx context.Context, payload string) error
	HandleGSheetDeleteJob(ctx context.Context, payload string) error
	HandleGSheetHeaderJob(ctx context.Context, payload string) error
	HandleGSheetRebuildJob(ctx context.Context, payload string) error

	// --- boot / periodic sweep ---
	SweepStaleDrafts(ctx context.Context) error
	SweepOrphanUploads(ctx context.Context) error
}
