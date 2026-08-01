package role_service

import (
	"context"
	"strings"

	"fsldk-api/base/apperror"
	"fsldk-api/modules/role/role_dto"
	"fsldk-api/modules/role/role_model"
	"fsldk-api/modules/role/role_repository"
)

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct{ repo role_repository.Repository }

// NewService membuat Service role.
func NewService(repo role_repository.Repository) Service { return &ServiceImpl{repo: repo} }

func (s *ServiceImpl) toResponse(ctx context.Context, r role_model.Role, withPerms bool) role_dto.Response {
	resp := role_dto.Response{
		RoleID:          r.RoleID,
		RoleName:        r.RoleName,
		RoleDescription: r.RoleDescription.String,
		IsSystemRole:    r.IsSystemRole,
		IsActive:        r.IsActive,
		UserCount:       r.UserCount,
		Permissions:     []string{},
		PermissionIDs:   []int64{},
	}
	if withPerms {
		if names, err := s.repo.PermissionNames(ctx, r.RoleID); err == nil && names != nil {
			resp.Permissions = names
		}
		if ids, err := s.repo.PermissionIDs(ctx, r.RoleID); err == nil && ids != nil {
			resp.PermissionIDs = ids
		}
	}
	return resp
}

func (s *ServiceImpl) List(ctx context.Context, search string) ([]role_dto.Response, error) {
	roles, err := s.repo.ListWithUserCount(ctx, search)
	if err != nil {
		return nil, apperror.Internal("")
	}
	out := make([]role_dto.Response, 0, len(roles))
	for _, r := range roles {
		out = append(out, s.toResponse(ctx, r, true))
	}
	return out, nil
}

func (s *ServiceImpl) Get(ctx context.Context, id int64) (role_dto.Response, error) {
	r, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return role_dto.Response{}, apperror.NotFound("Role tidak ditemukan")
	}
	n, _ := s.repo.CountUsers(ctx, id)
	r.UserCount = n
	return s.toResponse(ctx, r, true), nil
}

func (s *ServiceImpl) Create(ctx context.Context, req role_dto.CreateRequest) (role_dto.Response, error) {
	id, err := s.repo.Create(ctx, strings.TrimSpace(req.RoleName), strings.TrimSpace(req.RoleDescription))
	if err != nil {
		return role_dto.Response{}, apperror.Conflict("Nama role sudah digunakan")
	}
	return s.Get(ctx, id)
}

func (s *ServiceImpl) Update(ctx context.Context, id int64, req role_dto.UpdateRequest) (role_dto.Response, error) {
	r, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return role_dto.Response{}, apperror.NotFound("Role tidak ditemukan")
	}
	if r.IsSystemRole && r.RoleName != strings.TrimSpace(req.RoleName) {
		return role_dto.Response{}, apperror.Unprocessable("Nama role bawaan sistem tidak dapat diubah")
	}
	if err := s.repo.Update(ctx, id, strings.TrimSpace(req.RoleName), strings.TrimSpace(req.RoleDescription), req.IsActive); err != nil {
		return role_dto.Response{}, apperror.Internal("")
	}
	return s.Get(ctx, id)
}

func (s *ServiceImpl) Delete(ctx context.Context, id int64) error {
	r, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return apperror.NotFound("Role tidak ditemukan")
	}
	if r.IsSystemRole {
		return apperror.Unprocessable("Role bawaan sistem tidak dapat dihapus")
	}
	n, _ := s.repo.CountUsers(ctx, id)
	if n > 0 {
		return apperror.Unprocessable("Role masih digunakan oleh pengguna dan tidak dapat dihapus")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return apperror.Internal("")
	}
	return nil
}

func (s *ServiceImpl) SetPermissions(ctx context.Context, id int64, ids []int64) (role_dto.Response, error) {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return role_dto.Response{}, apperror.NotFound("Role tidak ditemukan")
	}
	if err := s.repo.SetPermissions(ctx, id, ids); err != nil {
		return role_dto.Response{}, apperror.Internal("")
	}
	return s.Get(ctx, id)
}

func (s *ServiceImpl) Users(ctx context.Context, id int64) ([]role_model.RoleUser, error) {
	users, err := s.repo.UsersByRole(ctx, id)
	if err != nil {
		return nil, apperror.Internal("")
	}
	if users == nil {
		users = []role_model.RoleUser{}
	}
	return users, nil
}
