package role

import (
	"context"
	"strings"

	"fsldk-api/base/apperror"
)

// Response adalah representasi role untuk API.
type Response struct {
	RoleID          int64    `json:"roleID"`
	RoleName        string   `json:"roleName"`
	RoleDescription string   `json:"roleDescription"`
	IsSystemRole    bool     `json:"isSystemRole"`
	IsActive        bool     `json:"isActive"`
	UserCount       int      `json:"userCount"`
	Permissions     []string `json:"permissions"`
	PermissionIDs   []int64  `json:"permissionIDs"`
}

// CreateRequest adalah body membuat role.
type CreateRequest struct {
	RoleName        string `json:"roleName" validate:"required,min=2,max=100"`
	RoleDescription string `json:"roleDescription" validate:"max=255"`
}

// UpdateRequest adalah body memperbarui role.
type UpdateRequest struct {
	RoleName        string `json:"roleName" validate:"required,min=2,max=100"`
	RoleDescription string `json:"roleDescription" validate:"max=255"`
	IsActive        bool   `json:"isActive"`
}

// SetPermissionsRequest adalah body menetapkan permission ke role.
type SetPermissionsRequest struct {
	PermissionIDs []int64 `json:"permissionIDs"`
}

// Service memuat logika bisnis role.
type Service struct{ repo Repository }

// NewService membuat Service role.
func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) toResponse(ctx context.Context, r Role, withPerms bool) Response {
	resp := Response{
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

// List mengembalikan daftar role beserta jumlah pengguna & permission.
func (s *Service) List(ctx context.Context, search string) ([]Response, error) {
	roles, err := s.repo.ListWithUserCount(ctx, search)
	if err != nil {
		return nil, apperror.Internal("")
	}
	out := make([]Response, 0, len(roles))
	for _, r := range roles {
		out = append(out, s.toResponse(ctx, r, true))
	}
	return out, nil
}

// Get mengembalikan satu role beserta permission-nya.
func (s *Service) Get(ctx context.Context, id int64) (Response, error) {
	r, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return Response{}, apperror.NotFound("Role tidak ditemukan")
	}
	n, _ := s.repo.CountUsers(ctx, id)
	r.UserCount = n
	return s.toResponse(ctx, r, true), nil
}

// Create membuat role baru.
func (s *Service) Create(ctx context.Context, req CreateRequest) (Response, error) {
	id, err := s.repo.Create(ctx, strings.TrimSpace(req.RoleName), strings.TrimSpace(req.RoleDescription))
	if err != nil {
		return Response{}, apperror.Conflict("Nama role sudah digunakan")
	}
	return s.Get(ctx, id)
}

// Update memperbarui role (role sistem tetap boleh diubah deskripsi/permission,
// namun nama tidak boleh dikosongkan).
func (s *Service) Update(ctx context.Context, id int64, req UpdateRequest) (Response, error) {
	r, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return Response{}, apperror.NotFound("Role tidak ditemukan")
	}
	if r.IsSystemRole && r.RoleName != strings.TrimSpace(req.RoleName) {
		return Response{}, apperror.Unprocessable("Nama role bawaan sistem tidak dapat diubah")
	}
	if err := s.repo.Update(ctx, id, strings.TrimSpace(req.RoleName), strings.TrimSpace(req.RoleDescription), req.IsActive); err != nil {
		return Response{}, apperror.Internal("")
	}
	return s.Get(ctx, id)
}

// Delete menghapus role (tidak boleh role sistem / masih dipakai pengguna).
func (s *Service) Delete(ctx context.Context, id int64) error {
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

// SetPermissions menetapkan daftar permission pada role.
func (s *Service) SetPermissions(ctx context.Context, id int64, ids []int64) (Response, error) {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return Response{}, apperror.NotFound("Role tidak ditemukan")
	}
	if err := s.repo.SetPermissions(ctx, id, ids); err != nil {
		return Response{}, apperror.Internal("")
	}
	return s.Get(ctx, id)
}

// Users mengembalikan daftar pengguna pemilik sebuah role.
func (s *Service) Users(ctx context.Context, id int64) ([]RoleUser, error) {
	users, err := s.repo.UsersByRole(ctx, id)
	if err != nil {
		return nil, apperror.Internal("")
	}
	if users == nil {
		users = []RoleUser{}
	}
	return users, nil
}
