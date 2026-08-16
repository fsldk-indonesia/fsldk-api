// Package organization_service memuat logika bisnis modul organization,
// termasuk resolusi cakupan akses berjenjang (LDK/Puskomda/Puskomnas).
package organization_service

import (
	"context"

	"fsldk-api/base/dto"
	"fsldk-api/modules/organization/organization_dto"
)

// CallerScope menampung identitas & scope organisasi pengguna pemanggil,
// diambil dari klaim token (lihat base/appctx).
type CallerScope struct {
	UserID               int64
	OrganizationID       *int64
	OrganizationTypeCode string
	WildcardTierAccess   string
}

// Service adalah kontrak logika bisnis organisasi.
type Service interface {
	// IsAccessible memenuhi kontrak middlewares.OrgScopeLoader.
	IsAccessible(ctx context.Context, callerOrganizationID *int64, callerOrganizationTypeCode, wildcardTierAccess string, targetOrganizationID int64) (bool, error)

	AccessibleList(ctx context.Context, caller CallerScope) ([]organization_dto.MeOrganization, error)
	List(ctx context.Context, caller CallerScope, q dto.ListQuery, typeFilter string) ([]organization_dto.Response, int, error)
	Get(ctx context.Context, id int64) (organization_dto.Response, error)
	Children(ctx context.Context, id int64) ([]organization_dto.Response, error)
	Create(ctx context.Context, caller CallerScope, req organization_dto.CreateRequest) (organization_dto.Response, error)
	Update(ctx context.Context, id int64, req organization_dto.UpdateRequest, actorID int64) (organization_dto.Response, error)
	SetActive(ctx context.Context, caller CallerScope, id int64, active bool) error
}
