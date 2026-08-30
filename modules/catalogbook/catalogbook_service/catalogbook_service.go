// Package catalogbook_service holds catalogbook module business logic.
package catalogbook_service

import (
	"context"

	"fsldk-api/base/dto"
	"fsldk-api/modules/catalogbook/catalogbook_dto"
	"fsldk-api/modules/catalogbook/catalogbook_model"
)

// CommentCleaner is the narrow slice of comment_service.Service this module
// depends on — accepting an interface (not importing comment_service
// directly) avoids a hard/circular package dependency, per CLAUDE.md's "no
// gin.Context/SQL in services" layering rule and the Go idiom of accepting
// interfaces at the consumer.
type CommentCleaner interface {
	DeleteByContent(ctx context.Context, contentType string, contentID int64) error
}

// Service is the business logic contract for the book catalog.
type Service interface {
	PublicList(ctx context.Context, q dto.ListQuery, f catalogbook_dto.Filter, sort string) ([]catalogbook_model.CatalogBook, int, error)
	CMSList(ctx context.Context, q dto.ListQuery, f catalogbook_dto.Filter) ([]catalogbook_model.CatalogBook, int, error)
	PublicDetail(ctx context.Context, slug string) (catalogbook_model.CatalogBook, error)
	Get(ctx context.Context, id int64) (catalogbook_model.CatalogBook, error)

	Categories(ctx context.Context) ([]catalogbook_model.BookCategory, error)
	Languages(ctx context.Context) ([]catalogbook_model.Language, error)
	AuthorTypes(ctx context.Context) ([]catalogbook_model.AuthorType, error)
	AvailabilityTypes(ctx context.Context) ([]catalogbook_model.AvailabilityType, error)

	Create(ctx context.Context, req catalogbook_dto.Request, actorID int64) (catalogbook_model.CatalogBook, error)
	Update(ctx context.Context, id int64, req catalogbook_dto.Request, actorID int64) (catalogbook_model.CatalogBook, error)
	SetActive(ctx context.Context, id int64, isActive bool, actorID int64) error
	Delete(ctx context.Context, id int64) error

	Like(ctx context.Context, id int64) (int, error)
}
