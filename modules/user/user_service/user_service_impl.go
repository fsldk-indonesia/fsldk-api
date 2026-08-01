package user_service

import (
	"context"
	"database/sql"
	"strings"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/security"
	"fsldk-api/modules/user/user_dto"
	"fsldk-api/modules/user/user_model"
	"fsldk-api/modules/user/user_repository"
)

// sortColumns memetakan field sort yang diizinkan ke kolom database.
var sortColumns = map[string]string{
	"fullName":    "u.fullName",
	"email":       "u.email",
	"createdDate": "u.createdDate",
}

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct{ repo user_repository.Repository }

// NewService membuat Service pengguna.
func NewService(repo user_repository.Repository) Service { return &ServiceImpl{repo: repo} }

func (s *ServiceImpl) List(ctx context.Context, q dto.ListQuery, roleID int64) ([]user_dto.Response, int, error) {
	orderBy := q.OrderBy(sortColumns, "u.createdDate DESC")
	users, total, err := s.repo.List(ctx, q.Search, roleID, q.Limit, q.Offset(), orderBy)
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	out := make([]user_dto.Response, 0, len(users))
	for _, u := range users {
		out = append(out, user_dto.ToResponse(u))
	}
	return out, total, nil
}

func (s *ServiceImpl) Get(ctx context.Context, id int64) (user_dto.Response, error) {
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return user_dto.Response{}, apperror.NotFound("Pengguna tidak ditemukan")
	}
	return user_dto.ToResponse(u), nil
}

func (s *ServiceImpl) Create(ctx context.Context, req user_dto.CreateRequest, actorID int64) (user_dto.Response, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	exists, err := s.repo.ExistsByEmail(ctx, email)
	if err != nil {
		return user_dto.Response{}, apperror.Internal("")
	}
	if exists {
		return user_dto.Response{}, apperror.Conflict("Email sudah terdaftar")
	}
	hashed, err := security.HashPassword(req.Password)
	if err != nil {
		return user_dto.Response{}, apperror.Internal("")
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	id, err := s.repo.Create(ctx, user_model.CreateParams{
		RoleID:        req.RoleID,
		FullName:      strings.TrimSpace(req.FullName),
		Email:         email,
		Password:      sql.NullString{String: hashed, Valid: true},
		EmailVerified: true,
		CreatedBy:     sql.NullInt64{Int64: actorID, Valid: actorID > 0},
	})
	if err != nil {
		return user_dto.Response{}, apperror.Internal("Gagal membuat pengguna")
	}
	if !active {
		_ = s.repo.SetActive(ctx, id, false, actorID)
	}
	return s.Get(ctx, id)
}

func (s *ServiceImpl) Update(ctx context.Context, id int64, req user_dto.UpdateRequest, actorID int64) (user_dto.Response, error) {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return user_dto.Response{}, apperror.NotFound("Pengguna tidak ditemukan")
	}
	if err := s.repo.Update(ctx, id, strings.TrimSpace(req.FullName), req.RoleID, req.IsActive, actorID); err != nil {
		return user_dto.Response{}, apperror.Internal("")
	}
	return s.Get(ctx, id)
}

func (s *ServiceImpl) SetStatus(ctx context.Context, id int64, active bool, actorID int64) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return apperror.NotFound("Pengguna tidak ditemukan")
	}
	if err := s.repo.SetActive(ctx, id, active, actorID); err != nil {
		return apperror.Internal("")
	}
	return nil
}

func (s *ServiceImpl) ResetPassword(ctx context.Context, id int64) (string, error) {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return "", apperror.NotFound("Pengguna tidak ditemukan")
	}
	temp, err := security.RandomToken(6)
	if err != nil {
		return "", apperror.Internal("")
	}
	hashed, err := security.HashPassword(temp)
	if err != nil {
		return "", apperror.Internal("")
	}
	if err := s.repo.SetPassword(ctx, id, hashed, true); err != nil {
		return "", apperror.Internal("")
	}
	return temp, nil
}

func (s *ServiceImpl) Delete(ctx context.Context, id, actorID int64) error {
	if id == actorID {
		return apperror.Unprocessable("Anda tidak dapat menghapus akun Anda sendiri")
	}
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return apperror.NotFound("Pengguna tidak ditemukan")
	}
	if err := s.repo.SoftDelete(ctx, id, actorID); err != nil {
		return apperror.Internal("")
	}
	return nil
}
