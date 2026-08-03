// Package dashboard_repository adalah lapisan akses data modul dashboard.
package dashboard_repository

import (
	"context"

	"fsldk-api/modules/dashboard/dashboard_dto"
)

// Repository adalah kontrak akses data dashboard.
type Repository interface {
	Summary(ctx context.Context) (dashboard_dto.Summary, error)
}
