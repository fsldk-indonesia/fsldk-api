package statistic_service

import (
	"context"

	"fsldk-api/base/dto"
	"fsldk-api/constants"
	"fsldk-api/modules/statistic/statistic_dto"
	"fsldk-api/modules/statistic/statistic_repository"
)

type serviceImpl struct {
	repo statistic_repository.Repository
}

// NewService creates a new instance of statistic Service.
func NewService(repo statistic_repository.Repository) Service {
	return &serviceImpl{repo: repo}
}

func (s *serviceImpl) NetworkStats(ctx context.Context) (*statistic_dto.NetworkStatsResponse, error) {
	byType, err := s.repo.CountByType(ctx)
	if err != nil {
		return nil, err
	}
	byProvince, err := s.repo.CountByProvince(ctx)
	if err != nil {
		return nil, err
	}
	byLevel, err := s.repo.LevelDistribution(ctx)
	if err != nil {
		return nil, err
	}
	activeKader, err := s.repo.CountActiveKader(ctx)
	if err != nil {
		return nil, err
	}

	resp := &statistic_dto.NetworkStatsResponse{
		TotalActiveKader: activeKader,
		ByProvince:       byProvince,
		ByLevel:          byLevel,
	}
	for _, t := range byType {
		switch t.OrganizationTypeCode {
		case constants.OrgTypePuskomnas:
			resp.TotalPuskomnas = t.Count
		case constants.OrgTypePuskomda:
			resp.TotalPuskomda = t.Count
		case constants.OrgTypeLDK:
			resp.TotalLDK = t.Count
		}
	}
	return resp, nil
}

func (s *serviceImpl) Directory(ctx context.Context, q dto.ListQuery, organizationTypeCode, provinceName string) ([]statistic_dto.DirectoryEntry, int, error) {
	return s.repo.Directory(ctx, q, organizationTypeCode, provinceName)
}
