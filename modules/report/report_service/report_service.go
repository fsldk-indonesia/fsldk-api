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
}

// OrgScopeResolver menyediakan resolusi cakupan akses organisasi berjenjang.
type OrgScopeResolver interface {
	AccessibleOrganizationIDs(ctx context.Context, callerOrganizationID *int64, callerOrganizationTypeCode, wildcardTierAccess string) ([]int64, error)
}

// Service adalah kontrak logika bisnis report.
type Service interface {
	ExportSubmissions(ctx context.Context, caller CallerScope, filter report_dto.ExportFilter) (report_dto.ExportResult, error)
}
