// Package report_service memuat logika bisnis modul report: ekspor laporan
// submission (Excel/CSV) dengan cakupan data identik dashboard (Section 28).
package report_service

import (
	"context"

	"fsldk-api/modules/report/report_dto"
)

// CallerScope menampung identitas & scope organisasi pengguna pemanggil.
type CallerScope struct {
	UserID               int64
	OrganizationID       *int64
	OrganizationTypeCode string
	WildcardTierAccess   string
	// RequestedOrganizationID adalah organizationID eksplisit dari query
	// `organizationID` (org-switcher shell cms-ldk/cms-puskomda) — divalidasi
	// accessible via OrgScopeResolver.IsAccessible sebelum dipakai.
	RequestedOrganizationID *int64
}

// OrgScopeResolver menyediakan resolusi cakupan akses organisasi berjenjang.
type OrgScopeResolver interface {
	IsAccessible(ctx context.Context, callerOrganizationID *int64, callerOrganizationTypeCode, wildcardTierAccess string, targetOrganizationID int64) (bool, error)
	AccessibleOrganizationIDs(ctx context.Context, callerOrganizationID *int64, callerOrganizationTypeCode, wildcardTierAccess string) ([]int64, error)
	AccessibleOrganizationIDsForTarget(ctx context.Context, targetOrganizationID int64) ([]int64, error)
}

// Service adalah kontrak logika bisnis report.
type Service interface {
	ExportSubmissions(ctx context.Context, caller CallerScope, filter report_dto.ExportFilter) (report_dto.ExportResult, error)
}
