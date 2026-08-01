package user

import (
	"context"
	"database/sql"
	"strings"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/security"
)

// sortColumns memetakan field sort yang diizinkan ke kolom database.
var sortColumns = map[string]string{
	"fullName":    "u.fullName",
	"email":       "u.email",
	"createdDate": "u.createdDate",
}

// Service memuat logika bisnis pengguna.
type Service struct{ repo Repository }

// NewService membuat Service pengguna.
func NewService(repo Repository) *Service { return &Service{repo: repo} }

// List mengembalikan daftar pengguna dengan paginasi.
func (s *Service) List(ctx context.Context, q dto.ListQuery, roleID int64) ([]Response, int, error) {
	orderBy := q.OrderBy(sortColumns, "u.createdDate DESC")
	users, total, err := s.repo.List(ctx, q.Search, roleID, q.Limit, q.Offset(), orderBy)
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	out := make([]Response, 0, len(users))
	for _, u := range users {
		out = append(out, ToResponse(u))
	}
	return out, total, nil
}

// Get mengembalikan satu pengguna.
func (s *Service) Get(ctx context.Context, id int64) (Response, error) {
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return Response{}, apperror.NotFound("Pengguna tidak ditemukan")
	}
	return ToResponse(u), nil
}

// Create membuat pengguna baru (oleh admin). Akun langsung terverifikasi.
func (s *Service) Create(ctx context.Context, req CreateRequest, actorID int64) (Response, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	exists, err := s.repo.ExistsByEmail(ctx, email)
	if err != nil {
		return Response{}, apperror.Internal("")
	}
	if exists {
		return Response{}, apperror.Conflict("Email sudah terdaftar")
	}
	hashed, err := security.HashPassword(req.Password)
	if err != nil {
		return Response{}, apperror.Internal("")
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	id, err := s.repo.Create(ctx, CreateParams{
		RoleID:        req.RoleID,
		FullName:      strings.TrimSpace(req.FullName),
		Email:         email,
		Password:      sql.NullString{String: hashed, Valid: true},
		EmailVerified: true,
		CreatedBy:     sql.NullInt64{Int64: actorID, Valid: actorID > 0},
	})
	if err != nil {
		return Response{}, apperror.Internal("Gagal membuat pengguna")
	}
	if !active {
		_ = s.repo.SetActive(ctx, id, false, actorID)
	}
	return s.Get(ctx, id)
}

// Update memperbarui data pengguna.
func (s *Service) Update(ctx context.Context, id int64, req UpdateRequest, actorID int64) (Response, error) {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return Response{}, apperror.NotFound("Pengguna tidak ditemukan")
	}
	if err := s.repo.Update(ctx, id, strings.TrimSpace(req.FullName), req.RoleID, req.IsActive, actorID); err != nil {
		return Response{}, apperror.Internal("")
	}
	return s.Get(ctx, id)
}

// SetStatus mengaktifkan/menonaktifkan pengguna.
func (s *Service) SetStatus(ctx context.Context, id int64, active bool, actorID int64) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return apperror.NotFound("Pengguna tidak ditemukan")
	}
	if err := s.repo.SetActive(ctx, id, active, actorID); err != nil {
		return apperror.Internal("")
	}
	return nil
}

// ResetPassword mereset password pengguna & mengembalikan password sementara.
// Berlaku juga untuk akun Google (menambah jalur login lokal).
func (s *Service) ResetPassword(ctx context.Context, id int64) (string, error) {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return "", apperror.NotFound("Pengguna tidak ditemukan")
	}
	temp, err := security.RandomToken(6) // 12 karakter hex
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

// Delete menghapus pengguna (soft delete).
func (s *Service) Delete(ctx context.Context, id, actorID int64) error {
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
