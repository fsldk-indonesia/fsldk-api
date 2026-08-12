// Package article_service memuat logika bisnis modul article.
package article_service

import (
	"context"

	"fsldk-api/base/dto"
	"fsldk-api/modules/article/article_dto"
	"fsldk-api/modules/article/article_model"
)

// CommentCleaner is the narrow slice of comment_service.Service this module
// depends on — accepting an interface (not importing comment_service
// directly) avoids a hard/circular package dependency, per CLAUDE.md's "no
// gin.Context/SQL in services" layering rule and the Go idiom of accepting
// interfaces at the consumer.
type CommentCleaner interface {
	DeleteByContent(ctx context.Context, contentType string, contentID int64) error
}

// Service adalah kontrak logika bisnis artikel.
type Service interface {
	PublicList(ctx context.Context, q dto.ListQuery, categorySlug string) ([]article_model.Article, int, error)
	CMSList(ctx context.Context, q dto.ListQuery, status string, categoryID int64) ([]article_model.Article, int, error)
	PublicDetail(ctx context.Context, slug string) (article_model.Article, error)
	Get(ctx context.Context, id int64) (article_model.Article, error)
	Categories(ctx context.Context) ([]article_model.Category, error)
	Create(ctx context.Context, req article_dto.Request, authorID int64, canPublish bool) (article_model.Article, error)
	Update(ctx context.Context, id int64, req article_dto.Request, updatedBy int64) (article_model.Article, error)
	SetPublished(ctx context.Context, id int64, published bool, updatedBy int64) error
	Delete(ctx context.Context, id int64) error
}
