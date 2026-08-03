// Package dashboard_service memuat logika bisnis modul dashboard.
package dashboard_service

import (
	"context"

	"fsldk-api/modules/dashboard/dashboard_dto"
)

// Service adalah kontrak logika dashboard.
type Service interface {
	Summary(ctx context.Context) (dashboard_dto.Summary, error)
}
