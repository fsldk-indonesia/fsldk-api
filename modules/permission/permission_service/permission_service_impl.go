package permission_service

import (
	"context"

	"fsldk-api/modules/permission/permission_model"
	"fsldk-api/modules/permission/permission_repository"
)

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct {
	repo permission_repository.Repository
}

// NewService membuat Service.
func NewService(repo permission_repository.Repository) Service { return &ServiceImpl{repo: repo} }

// RolePermissions memenuhi middlewares.PermissionLoader.
func (s *ServiceImpl) RolePermissions(ctx context.Context, roleID int64) ([]string, error) {
	return s.repo.CodesByRole(ctx, roleID)
}

// Menu mengembalikan menu sidebar CMS untuk sebuah role.
func (s *ServiceImpl) Menu(ctx context.Context, roleID int64) ([]permission_model.MenuItem, error) {
	items, err := s.repo.MenuByRole(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []permission_model.MenuItem{}
	}
	return items, nil
}

// ListAll mengembalikan seluruh permission (untuk manajemen role).
func (s *ServiceImpl) ListAll(ctx context.Context) ([]permission_model.Permission, error) {
	return s.repo.ListAll(ctx)
}
