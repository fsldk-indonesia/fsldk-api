// Package statistic_service provides business logic for the public network
// statistics feature.
package statistic_service

import (
	"context"

	"fsldk-api/base/dto"
	"fsldk-api/modules/statistic/statistic_dto"
)

// Service defines operations backing the public "Statistik Jaringan" page.
type Service interface {
	NetworkStats(ctx context.Context) (*statistic_dto.NetworkStatsResponse, error)
	Directory(ctx context.Context, q dto.ListQuery, organizationTypeCode, provinceName string) ([]statistic_dto.DirectoryEntry, int, error)
}
