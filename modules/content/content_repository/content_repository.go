// Package content_repository adalah lapisan akses data modul content.
package content_repository

import (
	"context"
	"errors"

	"fsldk-api/modules/content/content_dto"
	"fsldk-api/modules/content/content_model"
)

// ErrNotFound dikembalikan bila data tidak ditemukan.
var ErrNotFound = errors.New("data tidak ditemukan")

// Repository adalah kontrak akses data konten & struktur organisasi.
type Repository interface {
	ListContent(ctx context.Context, activeOnly bool) ([]content_model.Content, error)
	GetContentByKey(ctx context.Context, key string) (content_model.Content, error)
	UpdateContent(ctx context.Context, key, title, body string, updatedBy int64) error

	ListOrg(ctx context.Context, activeOnly bool) ([]content_model.OrgMember, error)
	CreateOrg(ctx context.Context, m content_dto.OrgRequest, createdBy int64) (int64, error)
	UpdateOrg(ctx context.Context, id int64, m content_dto.OrgRequest) error
	DeleteOrg(ctx context.Context, id int64) error
}
