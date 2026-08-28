// Package catalogbook_repository is the data access layer for the catalogbook module (GORM).
package catalogbook_repository

import (
	"context"
	"errors"

	"fsldk-api/modules/catalogbook/catalogbook_dto"
	"fsldk-api/modules/catalogbook/catalogbook_model"
)

// ErrNotFound is returned when a book cannot be found.
var ErrNotFound = errors.New("book not found")

// Repository is the data access contract for the book catalog.
type Repository interface {
	List(ctx context.Context, f catalogbook_dto.Filter) ([]catalogbook_model.CatalogBook, int64, error)
	FindByID(ctx context.Context, id int64) (catalogbook_model.CatalogBook, error)
	FindBySlug(ctx context.Context, slug string) (catalogbook_model.CatalogBook, error)
	SlugExists(ctx context.Context, slug string, exceptID int64) (bool, error)
	Create(ctx context.Context, b catalogbook_model.CatalogBook, actorID int64) (int64, error)
	Update(ctx context.Context, id int64, b catalogbook_model.CatalogBook, actorID int64) error
	SetActive(ctx context.Context, id int64, isActive bool, actorID int64) error
	IncrementFavorite(ctx context.Context, id int64) (int, error)
	Delete(ctx context.Context, id int64) error

	Categories(ctx context.Context) ([]catalogbook_model.BookCategory, error)
	Languages(ctx context.Context) ([]catalogbook_model.Language, error)
	AuthorTypes(ctx context.Context) ([]catalogbook_model.AuthorType, error)
	AvailabilityTypes(ctx context.Context) ([]catalogbook_model.AvailabilityType, error)
}
