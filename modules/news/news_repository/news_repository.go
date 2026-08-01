// Package news_repository adalah lapisan akses data modul news.
package news_repository

import (
	"context"
	"errors"

	"fsldk-api/modules/news/news_model"
)

// ErrNotFound dikembalikan bila berita tidak ditemukan.
var ErrNotFound = errors.New("berita tidak ditemukan")

// Filter menampung parameter penyaringan daftar berita.
type Filter struct {
	Search        string
	CategorySlug  string
	CategoryID    int64
	PublishedOnly bool
	Status        string
	Limit         int
	Offset        int
	OrderBy       string
}

// Repository adalah kontrak akses data berita.
type Repository interface {
	List(ctx context.Context, f Filter) ([]news_model.News, int, error)
	FindByID(ctx context.Context, id int64) (news_model.News, error)
	FindBySlug(ctx context.Context, slug string) (news_model.News, error)
	Featured(ctx context.Context, limit int) ([]news_model.News, error)
	SlugExists(ctx context.Context, slug string, exceptID int64) (bool, error)
	Create(ctx context.Context, n news_model.News, authorID int64) (int64, error)
	Update(ctx context.Context, id int64, n news_model.News, updatedBy int64) error
	SetPublished(ctx context.Context, id int64, published bool, updatedBy int64) error
	SetFeatured(ctx context.Context, id int64, featured bool, updatedBy int64) error
	Delete(ctx context.Context, id int64) error
	IncrementView(ctx context.Context, id int64) error
	Categories(ctx context.Context) ([]news_model.Category, error)
}
