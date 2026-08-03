package dashboard_service

import (
	"context"

	"fsldk-api/base/apperror"
	"fsldk-api/modules/dashboard/dashboard_dto"
	"fsldk-api/modules/dashboard/dashboard_repository"
)

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct {
	repo dashboard_repository.Repository
}

// NewService membuat Service dashboard.
func NewService(repo dashboard_repository.Repository) Service { return &ServiceImpl{repo: repo} }

func (s *ServiceImpl) Summary(ctx context.Context) (dashboard_dto.Summary, error) {
	out, err := s.repo.Summary(ctx)
	if err != nil {
		return dashboard_dto.Summary{}, apperror.Internal("")
	}
	return out, nil
}
