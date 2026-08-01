package dashboard_service

import (
	"context"

	"fsldk-api/base/apperror"
	"fsldk-api/modules/dashboard/dashboard_dto"
	"fsldk-api/modules/dashboard/dashboard_repository"
)

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct{ repo dashboard_repository.Repository }

// NewService membuat Service dashboard.
func NewService(repo dashboard_repository.Repository) Service { return &ServiceImpl{repo: repo} }

func (s *ServiceImpl) Summary(ctx context.Context) (dashboard_dto.Summary, error) {
	out, err := s.repo.Summary(ctx)
	if err != nil {
		return dashboard_dto.Summary{}, apperror.Internal("")
	}
	return out, nil
}

func (s *ServiceImpl) Recent(ctx context.Context, limit int) ([]dashboard_dto.RecentNews, error) {
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	out, err := s.repo.RecentNews(ctx, limit)
	if err != nil {
		return nil, apperror.Internal("")
	}
	return out, nil
}
