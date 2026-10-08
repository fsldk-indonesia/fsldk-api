// Package article_repository adalah lapisan akses data modul article (GORM).
package article_repository

import (
	"context"
	"errors"

	"fsldk-api/modules/article/article_dto"
	"fsldk-api/modules/article/article_model"
)

// ErrNotFound dikembalikan bila artikel tidak ditemukan.
var ErrNotFound = errors.New("artikel tidak ditemukan")

// Repository adalah kontrak akses data artikel.
type Repository interface {
	List(ctx context.Context, f article_dto.Filter) ([]article_model.Article, int64, error)
	FindByID(ctx context.Context, id int64) (article_model.Article, error)
	FindBySlug(ctx context.Context, slug string) (article_model.Article, error)
	SlugExists(ctx context.Context, slug string, exceptID int64) (bool, error)
	Create(ctx context.Context, a article_model.Article, authorID int64) (int64, error)
	Update(ctx context.Context, id int64, a article_model.Article, updatedBy int64) error
	SetPublished(ctx context.Context, id int64, published bool, updatedBy int64) error
	Delete(ctx context.Context, id int64) error
	Categories(ctx context.Context) ([]article_model.Category, error)
	// DistinctYears returns the distinct publishedDate years currently in the
	// table (published articles only) — used to populate the public "Tahun
	// Publikasi" filter dropdown without ever offering an empty-result option.
	DistinctYears(ctx context.Context) ([]int, error)
	// DistinctWriters returns the distinct articleWriter values currently in
	// the table (published articles only) — used to populate the public
	// "Penulis" filter dropdown without ever offering an empty-result option.
	DistinctWriters(ctx context.Context) ([]string, error)
}
