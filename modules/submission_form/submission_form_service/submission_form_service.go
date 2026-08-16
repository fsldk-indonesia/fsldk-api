// Package submission_form_service memuat logika bisnis modul submission_form:
// form builder (CRUD form/version/section/field/option) dan aturan versioning
// DRAFT → PUBLISHED → ARCHIVED.
package submission_form_service

import (
	"context"

	"fsldk-api/modules/submission_form/submission_form_dto"
)

// Service adalah kontrak logika bisnis submission form engine.
type Service interface {
	ListForms(ctx context.Context) ([]submission_form_dto.FormResponse, error)
	CreateForm(ctx context.Context, req submission_form_dto.CreateFormRequest, actorID int64) (submission_form_dto.FormResponse, error)
	GetForm(ctx context.Context, formID int64) (submission_form_dto.FormDetailResponse, error)

	CreateVersion(ctx context.Context, formID int64, req submission_form_dto.CreateVersionRequest, actorID int64) (submission_form_dto.VersionDetailResponse, error)
	GetVersion(ctx context.Context, versionID int64) (submission_form_dto.VersionDetailResponse, error)
	PublishVersion(ctx context.Context, versionID int64, actorID int64) (submission_form_dto.VersionDetailResponse, error)
	// GetPublishedByFormCode mengembalikan struktur version PUBLISHED aktif
	// milik sebuah form — dipakai LDK/Kader untuk merender form pengisian,
	// jadi tidak digerbang permission admin form builder (struktur form
	// bukan data sensitif, berbeda dengan jawaban submission).
	GetPublishedByFormCode(ctx context.Context, formCode string) (submission_form_dto.VersionDetailResponse, error)

	CreateSection(ctx context.Context, versionID int64, req submission_form_dto.CreateSectionRequest) (submission_form_dto.SectionResponse, error)
	UpdateSection(ctx context.Context, sectionID int64, req submission_form_dto.UpdateSectionRequest) (submission_form_dto.SectionResponse, error)
	DeleteSection(ctx context.Context, sectionID int64) error

	CreateField(ctx context.Context, sectionID int64, req submission_form_dto.CreateFieldRequest) (submission_form_dto.FieldResponse, error)
	UpdateField(ctx context.Context, fieldID int64, req submission_form_dto.UpdateFieldRequest) (submission_form_dto.FieldResponse, error)
	DeleteField(ctx context.Context, fieldID int64) error

	CreateOption(ctx context.Context, fieldID int64, req submission_form_dto.CreateOptionRequest) (submission_form_dto.OptionResponse, error)
	UpdateOption(ctx context.Context, optionID int64, req submission_form_dto.UpdateOptionRequest) (submission_form_dto.OptionResponse, error)
	DeleteOption(ctx context.Context, optionID int64) error
}
