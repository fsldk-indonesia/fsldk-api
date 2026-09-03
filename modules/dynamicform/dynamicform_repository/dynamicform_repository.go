// Package dynamicform_repository is the dynamicform data-access layer (GORM).
package dynamicform_repository

import (
	"context"
	"errors"
	"time"

	"fsldk-api/modules/dynamicform/dynamicform_dto"
	"fsldk-api/modules/dynamicform/dynamicform_model"

	"gorm.io/gorm"
)

// ErrNotFound is returned when a form/field/submission row cannot be found.
var ErrNotFound = errors.New("dynamicform: not found")

// SubmitData is the fully-resolved input of one submit transaction.
type SubmitData struct {
	FormID           int64
	FormVersion      int
	RespondentEmail  string
	RespondentName   *string
	RespondentUserID *int64
	IPAddress        string
	UserAgent        string
	IsValid          bool
	// Answers maps fieldID -> stored value (checkbox already JSON-encoded).
	Answers map[int64]string
	// Files maps fieldID -> file row to insert (fileURL already on disk).
	Files map[int64]dynamicform_model.File
	// DeleteDraftUserID, when non-nil, drops that user's draft in the same tx.
	DeleteDraftUserID *int64
}

// EditData is the resolved input of one CMS edit-response transaction.
type EditData struct {
	FormID          int64
	SubmissionID    int64
	RespondentEmail string
	RespondentName  *string
	// Answers maps fieldID -> new value for non-file fields (upsert).
	Answers map[int64]string
	// ReplacedFiles maps fieldID -> replacement file row; the old file row and
	// its answer are removed first.
	ReplacedFiles map[int64]dynamicform_model.File
}

// ValueCount is one GROUP BY answerValue bucket.
type ValueCount struct {
	Value string
	Count int
}

// Repository is the dynamicform data-access contract.
type Repository interface {
	// --- forms ---
	GetByID(ctx context.Context, id int64) (dynamicform_model.Form, error)
	GetBySlug(ctx context.Context, slug string) (dynamicform_model.Form, error)
	SlugExists(ctx context.Context, slug string, exceptID int64) (bool, error)
	CreateForm(ctx context.Context, values map[string]any) (int64, error)
	UpdateForm(ctx context.Context, id int64, values map[string]any) error
	SoftDeleteForm(ctx context.Context, id int64) error
	// PurgeFormChildren hard-deletes every child row (submissions+answers+files,
	// drafts, fields, sections, collaborators) in FK-safe order and returns the
	// file URLs that were referenced, for best-effort disk cleanup.
	PurgeFormChildren(ctx context.Context, formID int64) ([]string, error)
	ListForms(ctx context.Context, f dynamicform_dto.FormFilter) ([]dynamicform_model.Form, int64, []int, error)

	// --- collaborators ---
	ListCollaborators(ctx context.Context, formID int64) ([]dynamicform_model.Collaborator, error)
	ReplaceCollaborators(ctx context.Context, formID int64, rows []dynamicform_dto.CollaboratorInput) error
	IsCollaborator(ctx context.Context, formID, userID int64, roles ...string) (bool, error)

	// --- sections & fields ---
	ListSections(ctx context.Context, formID int64) ([]dynamicform_model.Section, error)
	ListFields(ctx context.Context, formID int64, activeOnly bool) ([]dynamicform_model.Field, error)
	GetField(ctx context.Context, formID, fieldID int64) (dynamicform_model.Field, error)
	AddField(ctx context.Context, formID int64, values map[string]any) (int64, error)
	UpdateField(ctx context.Context, formID, fieldID int64, values map[string]any) error
	SoftDeleteField(ctx context.Context, formID, fieldID int64) error
	ReorderFields(ctx context.Context, formID int64, order []int64) error

	// --- submissions ---
	CountSubmissionsSince(ctx context.Context, formID int64, ip string, since time.Time) (int, error)
	OldestSubmissionSince(ctx context.Context, formID int64, ip string, since time.Time) (time.Time, bool, error)
	HasSubmitted(ctx context.Context, formID int64, email string, userID *int64) (bool, error)
	Submit(ctx context.Context, in SubmitData) (int64, error)
	ListSubmissions(ctx context.Context, formID int64, f dynamicform_dto.SubmissionFilter) ([]dynamicform_model.Submission, int64, error)
	AllSubmissionsAsc(ctx context.Context, formID int64) ([]dynamicform_model.Submission, error)
	GetSubmission(ctx context.Context, formID, submissionID int64) (dynamicform_model.Submission, error)
	AnswersFor(ctx context.Context, submissionIDs []int64) (map[int64]map[int64]string, error)
	FilesFor(ctx context.Context, submissionIDs []int64) (map[int64][]dynamicform_model.File, error)
	EditSubmission(ctx context.Context, in EditData) error
	DeleteSubmission(ctx context.Context, formID, submissionID int64) ([]string, *int, error)
	DeleteAllSubmissions(ctx context.Context, formID int64) ([]string, error)
	SetGsheetRowIndex(ctx context.Context, submissionID int64, rowIndex int) error
	DecrementRowIndexesAfter(ctx context.Context, formID int64, rowIndex int) error
	TouchGsheetSync(ctx context.Context, formID int64, syncErr string) error

	// --- drafts ---
	GetDraft(ctx context.Context, formID, userID int64) (dynamicform_model.Draft, bool, error)
	MutateDraft(ctx context.Context, formID, userID int64, mutate func(current map[string]any) (map[string]any, error)) error
	DeleteDraft(ctx context.Context, formID, userID int64) error
	StaleDrafts(ctx context.Context, olderThan time.Time) ([]dynamicform_model.Draft, error)
	DeleteDraftByID(ctx context.Context, draftID int64) error

	// --- analytics ---
	SubmissionsPerDay(ctx context.Context, formID int64, since time.Time) (map[string]int, error)
	ValidCounts(ctx context.Context, formID int64) (valid int, invalid int, err error)
	TotalFiles(ctx context.Context, formID int64) (int, error)
	RecentSubmissions(ctx context.Context, formID int64, limit int) ([]dynamicform_model.Submission, error)
	AnswerValueCounts(ctx context.Context, formID, fieldID int64) ([]ValueCount, error)

	// --- sweep ---
	ReferencedFileURLs(ctx context.Context) (map[string]struct{}, error)

	// --- concurrency ---
	WithAdvisoryLock(ctx context.Context, key string, timeoutSeconds int, fn func() error) error

	// DB exposes the handle for advisory-lock helpers in the gsheet syncer.
	DB() *gorm.DB
}
