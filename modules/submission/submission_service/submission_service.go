// Package submission_service memuat logika bisnis modul submission: draft,
// simpan jawaban, submit (dengan validasi Section 25), dan pembatalan.
package submission_service

import (
	"context"

	"fsldk-api/base/dto"
	"fsldk-api/modules/submission/submission_dto"
)

// CallerScope menampung identitas & scope organisasi pengguna pemanggil.
type CallerScope struct {
	UserID               int64
	OrganizationID       *int64
	OrganizationTypeCode string
	WildcardTierAccess   string
}

// OrgScopeResolver menyediakan resolusi cakupan akses organisasi berjenjang.
// Diimplementasikan oleh modul organization.
type OrgScopeResolver interface {
	IsAccessible(ctx context.Context, callerOrganizationID *int64, callerOrganizationTypeCode, wildcardTierAccess string, targetOrganizationID int64) (bool, error)
	AccessibleOrganizationIDs(ctx context.Context, callerOrganizationID *int64, callerOrganizationTypeCode, wildcardTierAccess string) ([]int64, error)
}

// Service adalah kontrak logika bisnis submission.
type Service interface {
	Create(ctx context.Context, caller CallerScope, req submission_dto.CreateRequest) (submission_dto.Response, error)
	SaveAnswers(ctx context.Context, id int64, caller CallerScope, req submission_dto.SaveAnswersRequest) (submission_dto.DetailResponse, error)
	Submit(ctx context.Context, id int64, caller CallerScope) (submission_dto.Response, error)
	Cancel(ctx context.Context, id int64, caller CallerScope) error
	List(ctx context.Context, caller CallerScope, q dto.ListQuery, status string) ([]submission_dto.Response, int, error)
	Get(ctx context.Context, id int64, caller CallerScope) (submission_dto.DetailResponse, error)
}
