package content_service

import (
	"context"

	"fsldk-api/base/apperror"
	"fsldk-api/modules/content/content_dto"
	"fsldk-api/modules/content/content_model"
	"fsldk-api/modules/content/content_repository"
)

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct{ repo content_repository.Repository }

// NewService membuat Service content.
func NewService(repo content_repository.Repository) Service { return &ServiceImpl{repo: repo} }

func (s *ServiceImpl) ListContent(ctx context.Context, activeOnly bool) ([]content_model.Content, error) {
	data, err := s.repo.ListContent(ctx, activeOnly)
	if err != nil {
		return nil, apperror.Internal("")
	}
	if data == nil {
		data = []content_model.Content{}
	}
	return data, nil
}

func (s *ServiceImpl) GetContent(ctx context.Context, key string) (content_model.Content, error) {
	c, err := s.repo.GetContentByKey(ctx, key)
	if err != nil {
		return content_model.Content{}, apperror.NotFound("Konten tidak ditemukan")
	}
	return c, nil
}

func (s *ServiceImpl) UpdateContent(ctx context.Context, key string, req content_dto.ContentUpdateRequest, updatedBy int64) error {
	if err := s.repo.UpdateContent(ctx, key, req.ContentTitle, req.ContentBody, updatedBy); err != nil {
		return apperror.NotFound("Konten tidak ditemukan")
	}
	return nil
}

func (s *ServiceImpl) Profile(ctx context.Context) (map[string]string, error) {
	list, err := s.repo.ListContent(ctx, true)
	if err != nil {
		return nil, apperror.Internal("")
	}
	out := make(map[string]string, len(list))
	for _, c := range list {
		body := ""
		if c.ContentBody != nil {
			body = *c.ContentBody
		}
		out[c.ContentKey] = body
	}
	return out, nil
}

func (s *ServiceImpl) ListOrg(ctx context.Context, activeOnly bool) ([]content_model.OrgMember, error) {
	data, err := s.repo.ListOrg(ctx, activeOnly)
	if err != nil {
		return nil, apperror.Internal("")
	}
	if data == nil {
		data = []content_model.OrgMember{}
	}
	return data, nil
}

func (s *ServiceImpl) CreateOrg(ctx context.Context, req content_dto.OrgRequest, createdBy int64) (int64, error) {
	id, err := s.repo.CreateOrg(ctx, req, createdBy)
	if err != nil {
		return 0, apperror.Internal("")
	}
	return id, nil
}

func (s *ServiceImpl) UpdateOrg(ctx context.Context, id int64, req content_dto.OrgRequest) error {
	if err := s.repo.UpdateOrg(ctx, id, req); err != nil {
		return apperror.NotFound("Pengurus tidak ditemukan")
	}
	return nil
}

func (s *ServiceImpl) DeleteOrg(ctx context.Context, id int64) error {
	if err := s.repo.DeleteOrg(ctx, id); err != nil {
		return apperror.Internal("")
	}
	return nil
}
