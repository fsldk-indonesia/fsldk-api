// Package dynamicform_model holds the dynamicform module's DB row structs.
// Pure data: no methods, no functions. Nullable columns use pointers; *JSON
// columns are scanned as raw strings and (un)marshalled by the service.
package dynamicform_model

import "time"

// Form is one ms_dynamic_form row.
type Form struct {
	FormID                    int64      `gorm:"column:formID;primaryKey" json:"formID"`
	Title                     string     `gorm:"column:title" json:"title"`
	Slug                      string     `gorm:"column:slug" json:"slug"`
	Description               *string    `gorm:"column:description" json:"description"`
	HeaderImageURL            *string    `gorm:"column:headerImageUrl" json:"headerImageUrl"`
	Status                    string     `gorm:"column:status" json:"status"`
	Version                   int        `gorm:"column:version" json:"version"`
	MaxSubmission             *int       `gorm:"column:maxSubmission" json:"maxSubmission"`
	IsMultipleSubmit          bool       `gorm:"column:isMultipleSubmit" json:"isMultipleSubmit"`
	RequireLogin              bool       `gorm:"column:requireLogin" json:"requireLogin"`
	StartDate                 *time.Time `gorm:"column:startDate" json:"startDate"`
	EndDate                   *time.Time `gorm:"column:endDate" json:"endDate"`
	ConfirmationMessage       *string    `gorm:"column:confirmationMessage" json:"confirmationMessage"`
	RedirectURL               *string    `gorm:"column:redirectUrl" json:"redirectUrl"`
	NotifyEmailsJSON          *string    `gorm:"column:notifyEmailsJSON" json:"-"`
	SendConfirmationEmail     bool       `gorm:"column:sendConfirmationEmail" json:"sendConfirmationEmail"`
	RateLimitPerIP            int        `gorm:"column:rateLimitPerIP" json:"rateLimitPerIP"`
	RateLimitWindowMinutes    int        `gorm:"column:rateLimitWindowMinutes" json:"rateLimitWindowMinutes"`
	GsheetEnabled             bool       `gorm:"column:gsheetEnabled" json:"gsheetEnabled"`
	GsheetSpreadsheetID       *string    `gorm:"column:gsheetSpreadsheetID" json:"-"`
	GsheetSpreadsheetURL      *string    `gorm:"column:gsheetSpreadsheetURL" json:"gsheetSpreadsheetUrl"`
	GsheetTabName             string     `gorm:"column:gsheetTabName" json:"-"`
	GsheetLastSyncDate        *time.Time `gorm:"column:gsheetLastSyncDate" json:"gsheetLastSyncDate"`
	GsheetLastSyncError       *string    `gorm:"column:gsheetLastSyncError" json:"gsheetLastSyncError"`
	GdriveFormFolderID        *string    `gorm:"column:gdriveFormFolderID" json:"-"`
	GdriveAttachmentsFolderID *string    `gorm:"column:gdriveAttachmentsFolderID" json:"-"`
	GdriveAssetsFolderID      *string    `gorm:"column:gdriveAssetsFolderID" json:"-"`
	TotalSubmission           int        `gorm:"column:totalSubmission" json:"totalSubmission"`
	IsActive                  bool       `gorm:"column:isActive" json:"isActive"`
	CreatedDate               time.Time  `gorm:"column:createdDate" json:"createdDate"`
	CreatedBy                 *int64     `gorm:"column:createdBy" json:"createdBy"`
	CreatorName               string     `gorm:"column:creatorName;->" json:"creatorName"`
	UpdatedDate               *time.Time `gorm:"column:updatedDate" json:"updatedDate"`
}

// Field is one ms_dynamic_form_field row. Sections are not a table — a section
// is the run of fields between two `section_break` fields (techspec Part 2, K1).
// Section routing (forward-only jumps) rides inside fieldConfigJSON.
type Field struct {
	FieldID         int64   `gorm:"column:fieldID;primaryKey" json:"fieldID"`
	FormID          int64   `gorm:"column:formID" json:"formID"`
	FieldType       string  `gorm:"column:fieldType" json:"fieldType"`
	Label           string  `gorm:"column:label" json:"label"`
	Placeholder     *string `gorm:"column:placeholder" json:"placeholder"`
	HelpText        *string `gorm:"column:helpText" json:"helpText"`
	IsRequired      bool    `gorm:"column:isRequired" json:"isRequired"`
	IsSystemField   bool    `gorm:"column:isSystemField" json:"isSystemField"`
	SortOrder       int     `gorm:"column:sortOrder" json:"sortOrder"`
	OptionsJSON     *string `gorm:"column:optionsJSON" json:"-"`
	ValidationJSON  *string `gorm:"column:validationJSON" json:"-"`
	DefaultValue    *string `gorm:"column:defaultValue" json:"defaultValue"`
	FieldConfigJSON *string `gorm:"column:fieldConfigJSON" json:"-"`
	GdriveFolderID  *string `gorm:"column:gdriveFolderID" json:"-"`
	IsActive        bool    `gorm:"column:isActive" json:"-"`
}

// Submission is one tr_dynamic_form_submission row.
type Submission struct {
	SubmissionID     int64     `gorm:"column:submissionID;primaryKey" json:"submissionID"`
	FormID           int64     `gorm:"column:formID" json:"formID"`
	RespondentEmail  string    `gorm:"column:respondentEmail" json:"respondentEmail"`
	RespondentName   *string   `gorm:"column:respondentName" json:"respondentName"`
	RespondentUserID *int64    `gorm:"column:respondentUserID" json:"respondentUserID"`
	IPAddress        *string   `gorm:"column:ipAddress" json:"-"`
	UserAgent        *string   `gorm:"column:userAgent" json:"-"`
	IsValid          bool      `gorm:"column:isValid" json:"isValid"`
	FormVersion      int       `gorm:"column:formVersion" json:"formVersion"`
	GsheetRowIndex   *int      `gorm:"column:gsheetRowIndex" json:"-"`
	SubmittedDate    time.Time `gorm:"column:submittedDate" json:"submittedDate"`
}

// Answer is one tr_dynamic_form_answer row. answerValue holds a single value;
// checkbox/multiselect answers hold a JSON array string.
type Answer struct {
	AnswerID     int64   `gorm:"column:answerID;primaryKey" json:"answerID"`
	SubmissionID int64   `gorm:"column:submissionID" json:"submissionID"`
	FieldID      int64   `gorm:"column:fieldID" json:"fieldID"`
	AnswerValue  *string `gorm:"column:answerValue" json:"answerValue"`
}

// File is one tr_dynamic_form_file row.
type File struct {
	FileID           int64     `gorm:"column:fileID;primaryKey" json:"fileID"`
	SubmissionID     int64     `gorm:"column:submissionID" json:"submissionID"`
	FieldID          int64     `gorm:"column:fieldID" json:"fieldID"`
	FileURL          string    `gorm:"column:fileURL" json:"fileURL"`
	GdriveFileID     *string   `gorm:"column:gdriveFileID" json:"-"`
	OriginalFileName string    `gorm:"column:originalFileName" json:"originalFileName"`
	MimeType         *string   `gorm:"column:mimeType" json:"mimeType"`
	FileSizeKB       *int      `gorm:"column:fileSizeKB" json:"fileSizeKB"`
	CreatedDate      time.Time `gorm:"column:createdDate" json:"createdDate"`
}

// Draft is one tr_dynamic_form_draft row (one per formID+userID). answersJSON
// is a map "field_<id>" -> value, with staged file entries carrying a __file marker.
type Draft struct {
	DraftID     int64     `gorm:"column:draftID;primaryKey" json:"draftID"`
	FormID      int64     `gorm:"column:formID" json:"formID"`
	UserID      int64     `gorm:"column:userID" json:"userID"`
	AnswersJSON string    `gorm:"column:answersJSON" json:"-"`
	CreatedDate time.Time `gorm:"column:createdDate" json:"-"`
	UpdatedDate time.Time `gorm:"column:updatedDate" json:"-"`
}
