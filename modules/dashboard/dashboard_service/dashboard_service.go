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
}

// Service adalah kontrak logika dashboard.
type Service interface {
	Summary(ctx context.Context, caller CallerScope) (dashboard_dto.Summary, error)
}
