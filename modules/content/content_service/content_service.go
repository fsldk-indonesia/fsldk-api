// Package content_service memuat logika bisnis modul content.
package content_service

import (
	"context"

	"fsldk-api/modules/content/content_dto"
	"fsldk-api/modules/content/content_model"
)

// Service adalah kontrak logika bisnis konten & struktur organisasi.
type Service interface {
	ListContent(ctx context.Context, activeOnly bool) ([]content_model.Content, error)
	GetContent(ctx context.Context, key string) (content_model.Content, error)
	UpdateContent(ctx context.Context, key string, req content_dto.ContentUpdateRequest, updatedBy int64) error
	Profile(ctx context.Context) (map[string]string, error)
	ListOrg(ctx context.Context, activeOnly bool) ([]content_model.OrgMember, error)
	CreateOrg(ctx context.Context, req content_dto.OrgRequest, createdBy int64) (int64, error)
	UpdateOrg(ctx context.Context, id int64, req content_dto.OrgRequest) error
	DeleteOrg(ctx context.Context, id int64) error
}
