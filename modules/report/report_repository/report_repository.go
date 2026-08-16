// Package report_repository adalah lapisan akses data modul report.
package report_repository

import (
	"context"

	"fsldk-api/modules/report/report_dto"
)

// Repository adalah kontrak akses data laporan.
type Repository interface {
	SubmissionRows(ctx context.Context, formID int64, status string, organizationIDs []int64) ([]report_dto.SubmissionRow, error)
}
