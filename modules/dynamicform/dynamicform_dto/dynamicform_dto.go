// Package dynamicform_dto holds the dynamicform request/response DTOs.
// Pure data: no methods, no functions.
package dynamicform_dto

import "encoding/json"

// ---------------------------------------------------------------------------
// CMS — form metadata
// ---------------------------------------------------------------------------

// FormRequest is the create/update body for a form's metadata.
type FormRequest struct {
	Title                  string              `json:"title" validate:"required,min=3,max=255"`
	Description            *string             `json:"description" validate:"omitempty,max=5000"`
	MaxSubmission          *int                `json:"maxSubmission" validate:"omitempty,min=1"`
	IsMultipleSubmit       bool                `json:"isMultipleSubmit"`
	RequireLogin           bool                `json:"requireLogin"`
	StartDate              *string             `json:"startDate" validate:"omitempty,datetime=2006-01-02 15:04:05"`
	EndDate                *string             `json:"endDate" validate:"omitempty,datetime=2006-01-02 15:04:05"`
	ConfirmationMessage    *string             `json:"confirmationMessage" validate:"omitempty,max=2000"`
	RedirectURL            *string             `json:"redirectUrl" validate:"omitempty,url,max=500"`
	NotifyEmails           []string            `json:"notifyEmails" validate:"omitempty,dive,email"`
	SendConfirmationEmail  bool                `json:"sendConfirmationEmail"`
	RateLimitPerIP         int                 `json:"rateLimitPerIP" validate:"omitempty,min=1,max=100"`
	RateLimitWindowMinutes int                 `json:"rateLimitWindowMinutes" validate:"omitempty,min=1,max=1440"`
	GsheetEnabled          bool                `json:"gsheetEnabled"`
	Collaborators          []CollaboratorInput `json:"collaborators" validate:"omitempty,dive"`
}

// CollaboratorInput is one row of the collaborators editor.
type CollaboratorInput struct {
	UserID int64  `json:"userID" validate:"required"`
	Role   string `json:"role" validate:"required,oneof=editor manager"`
}

// ---------------------------------------------------------------------------
// Builder — fields
// ---------------------------------------------------------------------------

// FieldRequest is the add/update body for one field. The service loosens
// `label` for section_break/image and validates option-bearing types.
type FieldRequest struct {
	SectionID        *int64           `json:"sectionID" validate:"omitempty"`
	FieldType        string           `json:"fieldType" validate:"required,oneof=short_text long_text email number phone url date time datetime dropdown radio checkbox linear_scale rating file section_break paragraph image"`
	Label            string           `json:"label" validate:"omitempty,max=500"`
	Placeholder      *string          `json:"placeholder" validate:"omitempty,max=255"`
	HelpText         *string          `json:"helpText" validate:"omitempty,max=2000"`
	IsRequired       bool             `json:"isRequired"`
	Options          []FieldOption    `json:"options" validate:"omitempty,dive"`
	Validation       *FieldValidation `json:"validation"`
	DefaultValue     *string          `json:"defaultValue" validate:"omitempty,max=2000"`
	ConditionalLogic json.RawMessage  `json:"conditionalLogic"`
	FieldConfig      json.RawMessage  `json:"fieldConfig"`
	ImageURL         *string          `json:"imageURL" validate:"omitempty,url"`
}

// FieldOption is one choice for dropdown/radio/checkbox.
type FieldOption struct {
	Label string `json:"label" validate:"required,max=255"`
	Value string `json:"value" validate:"required,max=255"`
}

// FieldValidation is the structured validation config for a field.
type FieldValidation struct {
	Min           *int     `json:"min"`
	Max           *int     `json:"max"`
	Pattern       *string  `json:"pattern"`
	AcceptedTypes []string `json:"acceptedTypes"`
	MaxSizeKB     *int     `json:"maxSizeKB"`
}

// ReorderRequest carries the new fieldID order.
type ReorderRequest struct {
	Order []int64 `json:"order" validate:"required,min=1,dive,required"`
}

// StatusRequest carries a lifecycle transition target.
type StatusRequest struct {
	Status string `json:"status" validate:"required,oneof=published closed archived draft"`
}

// ---------------------------------------------------------------------------
// Public — draft autosave
// ---------------------------------------------------------------------------

// DraftRequest is the body of POST .../:slug/draft. Answers maps
// "field_<fieldID>" -> value. Files go through a separate endpoint.
type DraftRequest struct {
	Answers map[string]json.RawMessage `json:"answers"`
}

// EditSubmissionRequest is the JSON portion of the CMS edit-one-response
// multipart body. Answers maps "field_<fieldID>" -> value (string or array).
type EditSubmissionRequest struct {
	Answers map[string]json.RawMessage `json:"answers"`
}

// ---------------------------------------------------------------------------
// Filters
// ---------------------------------------------------------------------------

// FormFilter holds the CMS list filter (repository + service).
type FormFilter struct {
	Search   string
	Status   string
	DateFrom string
	DateTo   string
	MineOnly bool
	ActorID  int64
	Limit    int
	Offset   int
	OrderBy  string
}

// SubmissionFilter holds the rekap (responses) list filter.
type SubmissionFilter struct {
	Search    string
	ValidOnly bool
	DateFrom  string
	DateTo    string
	Limit     int
	Offset    int
}

// ---------------------------------------------------------------------------
// Responses
// ---------------------------------------------------------------------------

// FieldResponse is one field, JSON columns passed through raw for the frontend.
type FieldResponse struct {
	FieldID          int64           `json:"fieldID"`
	FormID           int64           `json:"formID"`
	SectionID        *int64          `json:"sectionID"`
	FieldType        string          `json:"fieldType"`
	Label            string          `json:"label"`
	Placeholder      *string         `json:"placeholder"`
	HelpText         *string         `json:"helpText"`
	IsRequired       bool            `json:"isRequired"`
	IsSystemField    bool            `json:"isSystemField"`
	SortOrder        int             `json:"sortOrder"`
	Options          json.RawMessage `json:"options"`
	Validation       json.RawMessage `json:"validation"`
	DefaultValue     *string         `json:"defaultValue"`
	ConditionalLogic json.RawMessage `json:"conditionalLogic"`
	FieldConfig      json.RawMessage `json:"fieldConfig"`
}

// SectionResponse is one section.
type SectionResponse struct {
	SectionID   int64   `json:"sectionID"`
	FormID      int64   `json:"formID"`
	Title       string  `json:"title"`
	Description *string `json:"description"`
	SortOrder   int     `json:"sortOrder"`
}

// CollaboratorResponse is one collaborator row for the metadata editor.
type CollaboratorResponse struct {
	UserID    int64  `json:"userID"`
	Role      string `json:"role"`
	UserName  string `json:"userName"`
	UserEmail string `json:"userEmail"`
}

// FormResponse is the full CMS view of a form (+ builder data).
type FormResponse struct {
	FormID                 int64                  `json:"formID"`
	Title                  string                 `json:"title"`
	Slug                   string                 `json:"slug"`
	Description            string                 `json:"description"`
	Status                 string                 `json:"status"`
	Version                int                    `json:"version"`
	MaxSubmission          *int                   `json:"maxSubmission"`
	IsMultipleSubmit       bool                   `json:"isMultipleSubmit"`
	RequireLogin           bool                   `json:"requireLogin"`
	StartDate              string                 `json:"startDate"`
	EndDate                string                 `json:"endDate"`
	ConfirmationMessage    string                 `json:"confirmationMessage"`
	RedirectURL            string                 `json:"redirectUrl"`
	NotifyEmails           []string               `json:"notifyEmails"`
	SendConfirmationEmail  bool                   `json:"sendConfirmationEmail"`
	RateLimitPerIP         int                    `json:"rateLimitPerIP"`
	RateLimitWindowMinutes int                    `json:"rateLimitWindowMinutes"`
	GsheetEnabled          bool                   `json:"gsheetEnabled"`
	GsheetSpreadsheetURL   string                 `json:"gsheetSpreadsheetUrl"`
	GsheetLastSyncDate     string                 `json:"gsheetLastSyncDate"`
	GsheetLastSyncError    string                 `json:"gsheetLastSyncError"`
	TotalSubmission        int                    `json:"totalSubmission"`
	IsActive               bool                   `json:"isActive"`
	CreatedDate            string                 `json:"createdDate"`
	CreatorName            string                 `json:"creatorName"`
	UpdatedDate            string                 `json:"updatedDate"`
	FieldCount             int                    `json:"fieldCount"`
	Collaborators          []CollaboratorResponse `json:"collaborators"`
	Sections               []SectionResponse      `json:"sections"`
	Fields                 []FieldResponse        `json:"fields"`
	PublicURL              string                 `json:"publicUrl"`
}

// PublicField is a field as served to the public renderer (no secret defaults).
type PublicField struct {
	FieldID          int64           `json:"fieldID"`
	SectionID        *int64          `json:"sectionID"`
	FieldType        string          `json:"fieldType"`
	Label            string          `json:"label"`
	Placeholder      *string         `json:"placeholder"`
	HelpText         *string         `json:"helpText"`
	IsRequired       bool            `json:"isRequired"`
	IsSystemField    bool            `json:"isSystemField"`
	SortOrder        int             `json:"sortOrder"`
	Options          json.RawMessage `json:"options"`
	Validation       json.RawMessage `json:"validation"`
	DefaultValue     *string         `json:"defaultValue"`
	ConditionalLogic json.RawMessage `json:"conditionalLogic"`
	FieldConfig      json.RawMessage `json:"fieldConfig"`
}

// PublicSection is a section for the public renderer.
type PublicSection struct {
	SectionID   int64   `json:"sectionID"`
	Title       string  `json:"title"`
	Description *string `json:"description"`
	SortOrder   int     `json:"sortOrder"`
}

// PublicFormResponse is what GET /public/dynamic-forms/:slug returns.
type PublicFormResponse struct {
	FormID           int64                      `json:"formID"`
	Title            string                     `json:"title"`
	Description      string                     `json:"description"`
	Slug             string                     `json:"slug"`
	Status           string                     `json:"status"`
	RequireLogin     bool                       `json:"requireLogin"`
	IsMultipleSubmit bool                       `json:"isMultipleSubmit"`
	Version          int                        `json:"version"`
	Sections         []PublicSection            `json:"sections"`
	Fields           []PublicField              `json:"fields"`
	PrefillEmail     string                     `json:"prefillEmail,omitempty"`
	DraftAnswers     map[string]json.RawMessage `json:"draftAnswers,omitempty"`
	IsPreview        bool                       `json:"isPreview"`
}

// SubmissionRow is one rekap table row. answers maps "field_<id>" -> display value.
type SubmissionRow struct {
	SubmissionID    int64             `json:"submissionID"`
	RespondentEmail string            `json:"respondentEmail"`
	RespondentName  string            `json:"respondentName"`
	IsValid         bool              `json:"isValid"`
	SubmittedDate   string            `json:"submittedDate"`
	Answers         map[string]string `json:"answers"`
}

// SubmissionFileRef is one uploaded file shown on the submission detail.
type SubmissionFileRef struct {
	FieldID          int64  `json:"fieldID"`
	FileURL          string `json:"fileURL"`
	OriginalFileName string `json:"originalFileName"`
}

// SubmissionDetail is one full submission for the CMS edit page.
type SubmissionDetail struct {
	SubmissionID    int64               `json:"submissionID"`
	FormID          int64               `json:"formID"`
	RespondentEmail string              `json:"respondentEmail"`
	RespondentName  string              `json:"respondentName"`
	IsValid         bool                `json:"isValid"`
	FormVersion     int                 `json:"formVersion"`
	SubmittedDate   string              `json:"submittedDate"`
	Answers         map[string]string   `json:"answers"`
	Fields          []FieldResponse     `json:"fields"`
	Files           []SubmissionFileRef `json:"files"`
}

// SubmitResult is the public submit reply.
type SubmitResult struct {
	Slug                string `json:"slug"`
	RedirectURL         string `json:"redirectUrl"`
	ConfirmationMessage string `json:"confirmationMessage"`
	IsMultipleSubmit    bool   `json:"isMultipleSubmit"`
}

// GSheetStatus is the reply of the gsheet connect/resync/disconnect endpoints.
type GSheetStatus struct {
	Enabled        bool   `json:"enabled"`
	SpreadsheetURL string `json:"spreadsheetUrl"`
	LastSyncDate   string `json:"lastSyncDate"`
	LastSyncError  string `json:"lastSyncError"`
}

// ---------------------------------------------------------------------------
// Analytics
// ---------------------------------------------------------------------------

// DayCount is one bucket of submissionsPerDay.
type DayCount struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// RecentSubmission is one row of the "10 latest" analytics list.
type RecentSubmission struct {
	SubmissionID    int64  `json:"submissionID"`
	RespondentEmail string `json:"respondentEmail"`
	RespondentName  string `json:"respondentName"`
	IsValid         bool   `json:"isValid"`
	SubmittedDate   string `json:"submittedDate"`
}

// ChartBucket is one label/count pair on a field chart.
type ChartBucket struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// FieldChart is the per-field aggregation.
type FieldChart struct {
	FieldID   int64         `json:"fieldID"`
	Label     string        `json:"label"`
	FieldType string        `json:"fieldType"`
	ChartType string        `json:"chartType"` // doughnut | bar | bar-horizontal
	Buckets   []ChartBucket `json:"buckets"`
}

// Analytics is the GET /:id/analytics payload.
type Analytics struct {
	SubmissionsPerDay []DayCount         `json:"submissionsPerDay"`
	ValidCount        int                `json:"validCount"`
	InvalidCount      int                `json:"invalidCount"`
	TotalFiles        int                `json:"totalFiles"`
	Recent            []RecentSubmission `json:"recent"`
	FieldCharts       []FieldChart       `json:"fieldCharts"`
}

// BulkDeleteRequest is the body of POST /bulk-delete.
type BulkDeleteRequest struct {
	IDs []int64 `json:"ids" validate:"required,min=1,dive,required"`
}

// BulkDeleteResult summarises a bulk delete.
type BulkDeleteResult struct {
	Deleted []int64 `json:"deleted"`
	Skipped []int64 `json:"skipped"`
}
