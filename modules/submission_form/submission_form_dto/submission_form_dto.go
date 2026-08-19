// Package submission_form_dto memuat DTO request/response modul submission_form.
package submission_form_dto

import (
	"encoding/json"
	"time"
)

// ---------- Response ----------

// FormResponse adalah representasi form template untuk API.
type FormResponse struct {
	FormID      int64     `json:"formID"`
	FormCode    string    `json:"formCode"`
	FormName    string    `json:"formName"`
	Description string    `json:"description,omitempty"`
	IsActive    bool      `json:"isActive"`
	CreatedDate time.Time `json:"createdDate"`
}

// VersionSummary adalah ringkasan version tanpa struktur section/field lengkap.
type VersionSummary struct {
	VersionID     int64      `json:"versionID"`
	VersionNumber int        `json:"versionNumber"`
	Status        string     `json:"status"`
	PublishedDate *time.Time `json:"publishedDate,omitempty"`
}

// FormDetailResponse adalah form beserta daftar ringkasan seluruh version-nya.
type FormDetailResponse struct {
	FormResponse
	Versions []VersionSummary `json:"versions"`
}

// OptionResponse adalah representasi pilihan field (SELECT/RADIO/CHECKBOX/MULTISELECT).
// Score hanya terisi bila field induk UseScoring && ScoringMethod=="AUTOMATIC".
type OptionResponse struct {
	OptionID    int64    `json:"optionID"`
	OptionValue string   `json:"optionValue"`
	OptionLabel string   `json:"optionLabel"`
	SortOrder   int      `json:"sortOrder"`
	IsActive    bool     `json:"isActive"`
	Score       *float64 `json:"score,omitempty"`
}

// FieldResponse adalah representasi satu field form. UseScoring/ScoringMethod/
// MinScore/MaxScore/Weight adalah konfigurasi scoring (enhancement Flexible
// Scoring) — lihat submission_form_model.Field.
type FieldResponse struct {
	FieldID              int64            `json:"fieldID"`
	SectionID            int64            `json:"sectionID"`
	FieldCode            string           `json:"fieldCode"`
	FieldLabel           string           `json:"fieldLabel"`
	FieldType            string           `json:"fieldType"`
	IsRequired           bool             `json:"isRequired"`
	SortOrder            int              `json:"sortOrder"`
	ValidationRule       json.RawMessage  `json:"validationRule,omitempty"`
	ConditionalOnFieldID *int64           `json:"conditionalOnFieldID,omitempty"`
	ConditionalRule      json.RawMessage  `json:"conditionalRule,omitempty"`
	HelpText             string           `json:"helpText,omitempty"`
	UseScoring           bool             `json:"useScoring"`
	ScoringMethod        string           `json:"scoringMethod,omitempty"`
	MinScore             *float64         `json:"minScore,omitempty"`
	MaxScore             *float64         `json:"maxScore,omitempty"`
	Weight               *float64         `json:"weight,omitempty"`
	Options              []OptionResponse `json:"options"`
}

// SectionResponse adalah representasi satu section beserta field-nya.
type SectionResponse struct {
	SectionID    int64           `json:"sectionID"`
	SectionCode  string          `json:"sectionCode"`
	SectionLabel string          `json:"sectionLabel"`
	SortOrder    int             `json:"sortOrder"`
	Description  string          `json:"description,omitempty"`
	Fields       []FieldResponse `json:"fields"`
}

// VersionDetailResponse adalah struktur lengkap satu version (untuk form builder & pengisian form).
type VersionDetailResponse struct {
	VersionID     int64             `json:"versionID"`
	FormID        int64             `json:"formID"`
	VersionNumber int               `json:"versionNumber"`
	Status        string            `json:"status"`
	PublishedDate *time.Time        `json:"publishedDate,omitempty"`
	Sections      []SectionResponse `json:"sections"`
}

// ---------- Request ----------

// CreateFormRequest adalah body membuat form template baru.
type CreateFormRequest struct {
	FormCode    string `json:"formCode" validate:"required,codeidentifier,min=2,max=50"`
	FormName    string `json:"formName" validate:"required,min=3,max=200"`
	Description string `json:"description" validate:"max=500"`
}

// CreateVersionRequest adalah body membuat version baru pada sebuah form.
// CloneFromVersionID opsional — bila diisi, seluruh section/field/option pada
// version tersebut disalin sebagai titik awal version baru.
type CreateVersionRequest struct {
	CloneFromVersionID *int64 `json:"cloneFromVersionID"`
}

// CreateSectionRequest adalah body menambah section pada sebuah version (harus berstatus DRAFT).
type CreateSectionRequest struct {
	SectionCode  string `json:"sectionCode" validate:"required,codeidentifier,min=2,max=50"`
	SectionLabel string `json:"sectionLabel" validate:"required,min=2,max=150"`
	SortOrder    int    `json:"sortOrder"`
	Description  string `json:"description" validate:"max=500"`
}

// UpdateSectionRequest adalah body memperbarui section.
type UpdateSectionRequest struct {
	SectionLabel string `json:"sectionLabel" validate:"required,min=2,max=150"`
	SortOrder    int    `json:"sortOrder"`
	Description  string `json:"description" validate:"max=500"`
}

// CreateFieldRequest adalah body menambah field pada sebuah section.
// UseScoring/ScoringMethod/MinScore/MaxScore/Weight opsional — divalidasi
// lebih lanjut di service layer (CreateField), bukan lewat tag validate,
// karena aturan lengkapnya saling bergantung (mis. wajib hanya saat
// UseScoring true, ScoringMethod=AUTOMATIC hanya untuk SELECT/RADIO).
type CreateFieldRequest struct {
	FieldCode            string          `json:"fieldCode" validate:"required,codeidentifier,min=2,max=50"`
	FieldLabel           string          `json:"fieldLabel" validate:"required,min=2,max=200"`
	FieldType            string          `json:"fieldType" validate:"required,oneof=TEXT TEXTAREA NUMBER DATE SELECT MULTISELECT RADIO CHECKBOX FILE_DOCUMENT FILE_IMAGE"`
	IsRequired           bool            `json:"isRequired"`
	SortOrder            int             `json:"sortOrder"`
	ValidationRule       json.RawMessage `json:"validationRule"`
	ConditionalOnFieldID *int64          `json:"conditionalOnFieldID"`
	ConditionalRule      json.RawMessage `json:"conditionalRule"`
	HelpText             string          `json:"helpText" validate:"max=500"`
	UseScoring           bool            `json:"useScoring"`
	ScoringMethod        string          `json:"scoringMethod" validate:"omitempty,oneof=AUTOMATIC MANUAL"`
	MinScore             *float64        `json:"minScore"`
	MaxScore             *float64        `json:"maxScore"`
	Weight               *float64        `json:"weight"`
}

// UpdateFieldRequest adalah body memperbarui field. FieldCode tidak dapat
// diubah setelah dibuat — kode ini menjadi acuan tetap jawaban tersimpan.
type UpdateFieldRequest struct {
	FieldLabel           string          `json:"fieldLabel" validate:"required,min=2,max=200"`
	FieldType            string          `json:"fieldType" validate:"required,oneof=TEXT TEXTAREA NUMBER DATE SELECT MULTISELECT RADIO CHECKBOX FILE_DOCUMENT FILE_IMAGE"`
	IsRequired           bool            `json:"isRequired"`
	SortOrder            int             `json:"sortOrder"`
	ValidationRule       json.RawMessage `json:"validationRule"`
	ConditionalOnFieldID *int64          `json:"conditionalOnFieldID"`
	ConditionalRule      json.RawMessage `json:"conditionalRule"`
	HelpText             string          `json:"helpText" validate:"max=500"`
	UseScoring           bool            `json:"useScoring"`
	ScoringMethod        string          `json:"scoringMethod" validate:"omitempty,oneof=AUTOMATIC MANUAL"`
	MinScore             *float64        `json:"minScore"`
	MaxScore             *float64        `json:"maxScore"`
	Weight               *float64        `json:"weight"`
}

// CreateOptionRequest adalah body menambah pilihan pada sebuah field. Score
// wajib diisi (divalidasi di service) hanya bila field induk menggunakan
// scoring otomatis.
type CreateOptionRequest struct {
	OptionValue string   `json:"optionValue" validate:"required,max=100"`
	OptionLabel string   `json:"optionLabel" validate:"required,max=200"`
	SortOrder   int      `json:"sortOrder"`
	Score       *float64 `json:"score"`
}

// UpdateOptionRequest adalah body memperbarui pilihan.
type UpdateOptionRequest struct {
	OptionValue string   `json:"optionValue" validate:"required,max=100"`
	OptionLabel string   `json:"optionLabel" validate:"required,max=200"`
	SortOrder   int      `json:"sortOrder"`
	IsActive    bool     `json:"isActive"`
	Score       *float64 `json:"score"`
}
