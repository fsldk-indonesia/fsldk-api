// Package news_service memuat logika bisnis modul news.
package news_service

import (
	"context"

	"fsldk-api/base/dto"
	"fsldk-api/modules/news/news_dto"
	"fsldk-api/modules/news/news_model"
)

// Service adalah kontrak logika bisnis berita.
type Service interface {
	PublicList(ctx context.Context, q dto.ListQuery, categorySlug string) ([]news_model.News, int, error)
	CMSList(ctx context.Context, q dto.ListQuery, status string, categoryID int64) ([]news_model.News, int, error)
	PublicDetail(ctx context.Context, slug string) (news_model.News, error)
	Get(ctx context.Context, id int64) (news_model.News, error)
	Featured(ctx context.Context, limit int) ([]news_model.News, error)
	Categories(ctx context.Context) ([]news_model.Category, error)
	Create(ctx context.Context, req news_dto.Request, authorID int64, canPublish bool) (news_model.News, error)
	Update(ctx context.Context, id int64, req news_dto.Request, updatedBy int64) (news_model.News, error)
	SetPublished(ctx context.Context, id int64, published bool, updatedBy int64) error
	SetFeatured(ctx context.Context, id int64, featured bool, updatedBy int64) error
	Delete(ctx context.Context, id int64) error
}
