// Package submission_form_model memuat entitas modul submission_form. Murni struct data.
package submission_form_model

import (
	"database/sql"
	"time"
)

// Form merepresentasikan satu baris ms_submission_form.
type Form struct {
	FormID      int64          `gorm:"column:formID;primaryKey"`
	FormCode    string         `gorm:"column:formCode"`
	FormName    string         `gorm:"column:formName"`
	Description sql.NullString `gorm:"column:description"`
	IsActive    bool           `gorm:"column:isActive"`
	CreatedDate time.Time      `gorm:"column:createdDate"`
}

// Version merepresentasikan satu baris ms_submission_form_version.
type Version struct {
	VersionID     int64        `gorm:"column:versionID;primaryKey"`
	FormID        int64        `gorm:"column:formID"`
	VersionNumber int          `gorm:"column:versionNumber"`
	Status        string       `gorm:"column:status"`
	PublishedDate sql.NullTime `gorm:"column:publishedDate"`
	CreatedDate   time.Time    `gorm:"column:createdDate"`
}

// Section merepresentasikan satu baris ms_submission_form_section.
type Section struct {
	SectionID    int64          `gorm:"column:sectionID;primaryKey"`
	VersionID    int64          `gorm:"column:versionID"`
	SectionCode  string         `gorm:"column:sectionCode"`
	SectionLabel string         `gorm:"column:sectionLabel"`
	SortOrder    int            `gorm:"column:sortOrder"`
	Description  sql.NullString `gorm:"column:description"`
}

// Field merepresentasikan satu baris ms_submission_form_field.
type Field struct {
	FieldID              int64          `gorm:"column:fieldID;primaryKey"`
	SectionID            int64          `gorm:"column:sectionID"`
	FieldCode            string         `gorm:"column:fieldCode"`
	FieldLabel           string         `gorm:"column:fieldLabel"`
	FieldType            string         `gorm:"column:fieldType"`
	IsRequired           bool           `gorm:"column:isRequired"`
	SortOrder            int            `gorm:"column:sortOrder"`
	ValidationRuleJSON   sql.NullString `gorm:"column:validationRuleJSON"`
	ConditionalOnFieldID sql.NullInt64  `gorm:"column:conditionalOnFieldID"`
	ConditionalRuleJSON  sql.NullString `gorm:"column:conditionalRuleJSON"`
	HelpText             sql.NullString `gorm:"column:helpText"`
}

// Option merepresentasikan satu baris ms_submission_form_field_option.
type Option struct {
	OptionID    int64  `gorm:"column:optionID;primaryKey"`
	FieldID     int64  `gorm:"column:fieldID"`
	OptionValue string `gorm:"column:optionValue"`
	OptionLabel string `gorm:"column:optionLabel"`
	SortOrder   int    `gorm:"column:sortOrder"`
	IsActive    bool   `gorm:"column:isActive"`
}

// FieldParams menampung data untuk membuat/memperbarui field.
type FieldParams struct {
	SectionID            int64
	FieldCode            string
	FieldLabel           string
	FieldType            string
	IsRequired           bool
	SortOrder            int
	ValidationRuleJSON   sql.NullString
	ConditionalOnFieldID sql.NullInt64
	ConditionalRuleJSON  sql.NullString
	HelpText             sql.NullString
}
