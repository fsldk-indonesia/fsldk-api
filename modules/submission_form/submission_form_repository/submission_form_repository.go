// Package submission_form_repository adalah lapisan akses data modul submission_form.
package submission_form_repository

import (
	"context"
	"database/sql"
	"errors"

	"fsldk-api/modules/submission_form/submission_form_model"
)

// ErrNotFound dikembalikan bila entitas tidak ditemukan.
var ErrNotFound = errors.New("data tidak ditemukan")

// Repository adalah kontrak akses data submission form engine.
type Repository interface {
	CreateForm(ctx context.Context, formCode, formName, description string, createdBy sql.NullInt64) (int64, error)
	ExistsFormCode(ctx context.Context, code string) (bool, error)
	ListForms(ctx context.Context) ([]submission_form_model.Form, error)
	FindFormByID(ctx context.Context, id int64) (submission_form_model.Form, error)

	CreateVersion(ctx context.Context, formID int64, versionNumber int, createdBy sql.NullInt64) (int64, error)
	MaxVersionNumber(ctx context.Context, formID int64) (int, error)
	FindVersionByID(ctx context.Context, id int64) (submission_form_model.Version, error)
	ListVersionsByForm(ctx context.Context, formID int64) ([]submission_form_model.Version, error)
	PublishVersion(ctx context.Context, id int64, publishedBy int64) error
	ArchiveOtherPublished(ctx context.Context, formID, exceptVersionID int64) error

	CreateSection(ctx context.Context, versionID int64, code, label string, sortOrder int, description sql.NullString) (int64, error)
	FindSectionByID(ctx context.Context, id int64) (submission_form_model.Section, error)
	UpdateSection(ctx context.Context, id int64, label string, sortOrder int, description sql.NullString) error
	DeleteSection(ctx context.Context, id int64) error
	ListSectionsByVersion(ctx context.Context, versionID int64) ([]submission_form_model.Section, error)

	CreateField(ctx context.Context, p submission_form_model.FieldParams) (int64, error)
	FindFieldByID(ctx context.Context, id int64) (submission_form_model.Field, error)
	UpdateField(ctx context.Context, id int64, p submission_form_model.FieldParams) error
	DeleteField(ctx context.Context, id int64) error
	ListFieldsByVersion(ctx context.Context, versionID int64) ([]submission_form_model.Field, error)

	CreateOption(ctx context.Context, fieldID int64, value, label string, sortOrder int) (int64, error)
	FindOptionByID(ctx context.Context, id int64) (submission_form_model.Option, error)
	UpdateOption(ctx context.Context, id int64, value, label string, sortOrder int, isActive bool) error
	DeleteOption(ctx context.Context, id int64) error
	ListOptionsByVersion(ctx context.Context, versionID int64) ([]submission_form_model.Option, error)

	VersionStatusBySectionID(ctx context.Context, sectionID int64) (string, error)
	VersionStatusByFieldID(ctx context.Context, fieldID int64) (string, error)
	VersionIDByFieldID(ctx context.Context, fieldID int64) (int64, error)

	// CloneVersionStructure menyalin seluruh section/field/option dari satu
	// version ke version lain (dipakai saat membuat version baru dari yang lama).
	CloneVersionStructure(ctx context.Context, fromVersionID, toVersionID int64) error
}
