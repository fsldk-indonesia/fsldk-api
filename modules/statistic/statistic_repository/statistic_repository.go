// Package statistic_repository provides read-only data access for the public
// network statistics feature.
package statistic_repository

import (
	"context"

	"fsldk-api/base/dto"
	"fsldk-api/modules/statistic/statistic_dto"
)

// Repository defines aggregate/read-only queries backing the public
// "Statistik Jaringan" endpoints. There is no Create/Update/Delete here —
// this module only reads data owned by the organization/submission modules.
type Repository interface {
	CountByType(ctx context.Context) ([]statistic_dto.TypeCount, error)
	CountByProvince(ctx context.Context) ([]statistic_dto.ProvinceCount, error)
	LevelDistribution(ctx context.Context) ([]statistic_dto.LevelCount, error)
	CountActiveKader(ctx context.Context) (int, error)
	Directory(ctx context.Context, q dto.ListQuery, organizationTypeCode, provinceName string) ([]statistic_dto.DirectoryEntry, int, error)
}
