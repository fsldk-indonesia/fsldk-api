// Package submission_repository adalah lapisan akses data modul submission.
package submission_repository

import (
	"context"
	"database/sql"
	"errors"

	"fsldk-api/modules/submission/submission_dto"
	"fsldk-api/modules/submission/submission_model"
)

// ErrNotFound dikembalikan bila entitas tidak ditemukan.
var ErrNotFound = errors.New("data tidak ditemukan")

// Repository adalah kontrak akses data submission.
type Repository interface {
	Create(ctx context.Context, p submission_model.CreateParams) (int64, error)
	FindByID(ctx context.Context, id int64) (submission_model.Submission, error)
	// FindByOwnerOrganization mencari submission ber-subjek ORGANIZATION milik
	// sebuah organisasi untuk form tertentu (maksimal satu per organisasi/form).
	FindByOwnerOrganization(ctx context.Context, organizationID, formID int64) (submission_model.Submission, error)
	// FindByUserAndForm mencari submission ber-subjek KADER milik seorang
	// pengguna untuk form tertentu (maksimal satu per pengguna/form).
	FindByUserAndForm(ctx context.Context, userID, formID int64) (submission_model.Submission, error)
	List(ctx context.Context, f submission_dto.ListFilter) ([]submission_model.Submission, int64, error)
	UpdateStatus(ctx context.Context, id int64, toStatus string, submittedDate sql.NullTime, updatedBy int64) error
	// TouchVersion menaikkan kolom version tanpa mengubah status — dipakai
	// saat menyimpan draft jawaban (Section 16: version naik tiap pembaruan).
	TouchVersion(ctx context.Context, id int64, updatedBy int64) error
	SetSubjectReference(ctx context.Context, id int64, subjectReferenceID int64) error

	UpsertAnswer(ctx context.Context, submissionID int64, p submission_model.AnswerParams) error
	ListAnswersBySubmission(ctx context.Context, submissionID int64) ([]submission_model.Answer, error)

	InsertStatusHistory(ctx context.Context, submissionID int64, fromStatus sql.NullString, toStatus string, actorUserID int64, actorOrganizationID sql.NullInt64, note sql.NullString) error
	ListStatusHistory(ctx context.Context, submissionID int64) ([]submission_model.StatusHistory, error)

	CreateKader(ctx context.Context, p submission_model.KaderParams) (int64, error)
	FindKaderBySubmission(ctx context.Context, submissionID int64) (submission_model.Kader, error)
	FindKaderByID(ctx context.Context, id int64) (submission_model.Kader, error)
	UpdateKaderStatus(ctx context.Context, id int64, status string, uniqueCode sql.NullString, issuedDate sql.NullTime) error
	ListKadersByOrganizations(ctx context.Context, organizationIDs []int64, status string) ([]submission_model.Kader, error)
	CountKaderCodesForOrgYear(ctx context.Context, codePrefix string) (int, error)

	CreateReview(ctx context.Context, p submission_model.ReviewParams) (int64, error)
	ListReviewsBySubmission(ctx context.Context, submissionID int64) ([]submission_model.Review, error)

	// FindCurrentLevelResult mencari baris tr_levelisasi_result dengan
	// isCurrent=1 milik sebuah organisasi (hasil resmi yang berlaku saat ini).
	FindCurrentLevelResult(ctx context.Context, organizationID int64) (submission_model.LevelisasiResult, error)
	FindLevelResultBySubmission(ctx context.Context, submissionID int64) (submission_model.LevelisasiResult, error)
	// ClearCurrentLevelResult menandai baris isCurrent=1 milik organisasi
	// (bila ada) menjadi isCurrent=0 — dipanggil sebelum insert siklus baru.
	ClearCurrentLevelResult(ctx context.Context, organizationID int64) error
	CreateLevelResult(ctx context.Context, submissionID, organizationID int64, levelCode, justificationNote string, establishedByUserID int64) (int64, error)
	UpdateLevelResult(ctx context.Context, resultID int64, levelCode, justificationNote string, establishedByUserID int64) error
	PublishLevelResult(ctx context.Context, resultID int64) error
	LevelLabel(ctx context.Context, levelCode string) (string, error)
}
