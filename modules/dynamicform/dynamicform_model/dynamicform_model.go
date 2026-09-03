// Package dynamicform_model holds the dynamicform module's DB row structs.
// Pure data: no methods, no functions. Nullable columns use pointers; *JSON
// columns are scanned as raw strings and (un)marshalled by the service.
package dynamicform_model

import "time"

// Form is one ms_dynamic_form row.
type Form struct {
	FormID                 int64      `gorm:"column:formID;primaryKey" json:"formID"`
	Title                  string     `gorm:"column:title" json:"title"`
	Slug                   string     `gorm:"column:slug" json:"slug"`
	Description            *string    `gorm:"column:description" json:"description"`
	Status                 string     `gorm:"column:status" json:"status"`
	Version                int        `gorm:"column:version" json:"version"`
	MaxSubmission          *int       `gorm:"column:maxSubmission" json:"maxSubmission"`
	IsMultipleSubmit       bool       `gorm:"column:isMultipleSubmit" json:"isMultipleSubmit"`
	RequireLogin           bool       `gorm:"column:requireLogin" json:"requireLogin"`
	StartDate              *time.Time `gorm:"column:startDate" json:"startDate"`
	EndDate                *time.Time `gorm:"column:endDate" json:"endDate"`
	ConfirmationMessage    *string    `gorm:"column:confirmationMessage" json:"confirmationMessage"`
	RedirectURL            *string    `gorm:"column:redirectUrl" json:"redirectUrl"`
	NotifyEmailsJSON       *string    `gorm:"column:notifyEmailsJSON" json:"-"`
	SendConfirmationEmail  bool       `gorm:"column:sendConfirmationEmail" json:"sendConfirmationEmail"`
	RateLimitPerIP         int        `gorm:"column:rateLimitPerIP" json:"rateLimitPerIP"`
	RateLimitWindowMinutes int        `gorm:"column:rateLimitWindowMinutes" json:"rateLimitWindowMinutes"`
	GsheetEnabled          bool       `gorm:"column:gsheetEnabled" json:"gsheetEnabled"`
	GsheetSpreadsheetID    *string    `gorm:"column:gsheetSpreadsheetID" json:"-"`
	GsheetSpreadsheetURL   *string    `gorm:"column:gsheetSpreadsheetURL" json:"gsheetSpreadsheetUrl"`
	GsheetTabName          string     `gorm:"column:gsheetTabName" json:"-"`
	GsheetLastSyncDate     *time.Time `gorm:"column:gsheetLastSyncDate" json:"gsheetLastSyncDate"`
	GsheetLastSyncError    *string    `gorm:"column:gsheetLastSyncError" json:"gsheetLastSyncError"`
	TotalSubmission        int        `gorm:"column:totalSubmission" json:"totalSubmission"`
	IsActive               bool       `gorm:"column:isActive" json:"isActive"`
	CreatedDate            time.Time  `gorm:"column:createdDate" json:"createdDate"`
	CreatedBy              *int64     `gorm:"column:createdBy" json:"createdBy"`
	CreatorName            string     `gorm:"column:creatorName;->" json:"creatorName"`
	UpdatedDate            *time.Time `gorm:"column:updatedDate" json:"updatedDate"`
}

// Section is one ms_dynamic_form_section row.
type Section struct {
	SectionID   int64      `gorm:"column:sectionID;primaryKey" json:"sectionID"`
	FormID      int64      `gorm:"column:formID" json:"formID"`
	Title       string     `gorm:"column:title" json:"title"`
	Description *string    `gorm:"column:description" json:"description"`
	SortOrder   int        `gorm:"column:sortOrder" json:"sortOrder"`
	IsActive    bool       `gorm:"column:isActive" json:"-"`
	CreatedDate time.Time  `gorm:"column:createdDate" json:"-"`
	UpdatedDate *time.Time `gorm:"column:updatedDate" json:"-"`
}

// Field is one ms_dynamic_form_field row.
type Field struct {
	FieldID              int64   `gorm:"column:fieldID;primaryKey" json:"fieldID"`
	FormID               int64   `gorm:"column:formID" json:"formID"`
	SectionID            *int64  `gorm:"column:sectionID" json:"sectionID"`
	FieldType            string  `gorm:"column:fieldType" json:"fieldType"`
	Label                string  `gorm:"column:label" json:"label"`
	Placeholder          *string `gorm:"column:placeholder" json:"placeholder"`
	HelpText             *string `gorm:"column:helpText" json:"helpText"`
	IsRequired           bool    `gorm:"column:isRequired" json:"isRequired"`
	IsSystemField        bool    `gorm:"column:isSystemField" json:"isSystemField"`
	SortOrder            int     `gorm:"column:sortOrder" json:"sortOrder"`
	OptionsJSON          *string `gorm:"column:optionsJSON" json:"-"`
	ValidationJSON       *string `gorm:"column:validationJSON" json:"-"`
	DefaultValue         *string `gorm:"column:defaultValue" json:"defaultValue"`
	ConditionalLogicJSON *string `gorm:"column:conditionalLogicJSON" json:"-"`
	FieldConfigJSON      *string `gorm:"column:fieldConfigJSON" json:"-"`
	IsActive             bool    `gorm:"column:isActive" json:"-"`
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

// Collaborator is one map_dynamic_form_collaborator row.
type Collaborator struct {
	CollaboratorID int64     `gorm:"column:collaboratorID;primaryKey" json:"collaboratorID"`
	FormID         int64     `gorm:"column:formID" json:"formID"`
	UserID         int64     `gorm:"column:userID" json:"userID"`
	Role           string    `gorm:"column:role" json:"role"`
	AddedDate      time.Time `gorm:"column:addedDate" json:"addedDate"`
	UserName       string    `gorm:"column:userName;->" json:"userName"`
	UserEmail      string    `gorm:"column:userEmail;->" json:"userEmail"`
}
