// Package dashboard_service memuat logika bisnis modul dashboard.
package dashboard_service

import (
	"context"

	"fsldk-api/modules/dashboard/dashboard_dto"
)

// CallerScope menampung identitas scope organisasi pengguna pemanggil.
type CallerScope struct {
	OrganizationID       *int64
	OrganizationTypeCode string
	WildcardTierAccess   string
	// RequestedOrganizationID adalah organizationID eksplisit dari query
	// `organizationID` (org-switcher shell cms-ldk/cms-puskomda/cms-puskomnas)
	// — divalidasi accessible via OrgScopeResolver.IsAccessible sebelum
	// menggantikan home org caller sebagai subjek ringkasan dashboard.
	RequestedOrganizationID *int64
}

// OrgScopeResolver menyediakan validasi & resolusi tipe organisasi target.
// Diimplementasikan oleh modul organization.
type OrgScopeResolver interface {
	IsAccessible(ctx context.Context, callerOrganizationID *int64, callerOrganizationTypeCode, wildcardTierAccess string, targetOrganizationID int64) (bool, error)
	TypeCodeByID(ctx context.Context, id int64) (string, error)
}

// Service adalah kontrak logika dashboard.
type Service interface {
	Summary(ctx context.Context, caller CallerScope) (dashboard_dto.Summary, error)
}
