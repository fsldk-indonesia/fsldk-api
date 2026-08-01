package content

import (
	"context"

	"fsldk-api/base/apperror"
)

// Service memuat logika bisnis konten & struktur organisasi.
type Service struct{ repo Repository }

// NewService membuat Service content.
func NewService(repo Repository) *Service { return &Service{repo: repo} }

// ListContent mengembalikan daftar konten (activeOnly untuk publik).
func (s *Service) ListContent(ctx context.Context, activeOnly bool) ([]Content, error) {
	data, err := s.repo.ListContent(ctx, activeOnly)
	if err != nil {
		return nil, apperror.Internal("")
	}
	if data == nil {
		data = []Content{}
	}
	return data, nil
}

// GetContent mengembalikan konten berdasarkan key.
func (s *Service) GetContent(ctx context.Context, key string) (Content, error) {
	c, err := s.repo.GetContentByKey(ctx, key)
	if err != nil {
		return Content{}, apperror.NotFound("Konten tidak ditemukan")
	}
	return c, nil
}

// UpdateContent memperbarui konten.
func (s *Service) UpdateContent(ctx context.Context, key string, req ContentUpdateRequest, updatedBy int64) error {
	if err := s.repo.UpdateContent(ctx, key, req.ContentTitle, req.ContentBody, updatedBy); err != nil {
		return apperror.NotFound("Konten tidak ditemukan")
	}
	return nil
}

// Profile mengembalikan konten Landing Page sebagai map key→body (untuk /public/profile).
func (s *Service) Profile(ctx context.Context) (map[string]string, error) {
	list, err := s.repo.ListContent(ctx, true)
	if err != nil {
		return nil, apperror.Internal("")
	}
	out := make(map[string]string, len(list))
	for _, c := range list {
		out[c.ContentKey] = c.ContentBody.String
	}
	return out, nil
}

// ListOrg mengembalikan struktur organisasi.
func (s *Service) ListOrg(ctx context.Context, activeOnly bool) ([]OrgMember, error) {
	data, err := s.repo.ListOrg(ctx, activeOnly)
	if err != nil {
		return nil, apperror.Internal("")
	}
	if data == nil {
		data = []OrgMember{}
	}
	return data, nil
}

// CreateOrg menambah pengurus.
func (s *Service) CreateOrg(ctx context.Context, req OrgRequest, createdBy int64) (int64, error) {
	id, err := s.repo.CreateOrg(ctx, req, createdBy)
	if err != nil {
		return 0, apperror.Internal("")
	}
	return id, nil
}

// UpdateOrg memperbarui pengurus.
func (s *Service) UpdateOrg(ctx context.Context, id int64, req OrgRequest) error {
	if err := s.repo.UpdateOrg(ctx, id, req); err != nil {
		return apperror.NotFound("Pengurus tidak ditemukan")
	}
	return nil
}

// DeleteOrg menghapus pengurus.
func (s *Service) DeleteOrg(ctx context.Context, id int64) error {
	if err := s.repo.DeleteOrg(ctx, id); err != nil {
		return apperror.Internal("")
	}
	return nil
}
