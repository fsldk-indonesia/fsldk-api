// Package goods_repository adalah lapisan akses data modul goods (GORM).
package goods_repository

import (
	"context"
	"errors"

	"fsldk-api/modules/goods/goods_dto"
	"fsldk-api/modules/goods/goods_model"
)

// ErrNotFound dikembalikan bila produk tidak ditemukan.
var ErrNotFound = errors.New("produk tidak ditemukan")

// ErrCategoryNotFound dikembalikan bila kategori tidak ditemukan.
var ErrCategoryNotFound = errors.New("kategori tidak ditemukan")

// Repository adalah kontrak akses data produk & kategori goods.
type Repository interface {
	List(ctx context.Context, f goods_dto.Filter) ([]goods_model.Goods, int64, error)
	FindByID(ctx context.Context, id int64) (goods_model.Goods, error)
	FindBySlug(ctx context.Context, slug string) (goods_model.Goods, error)
	SlugExists(ctx context.Context, slug string, exceptID int64) (bool, error)
	SKUExists(ctx context.Context, sku string, exceptID int64) (bool, error)
	Create(ctx context.Context, g goods_model.Goods, createdBy int64) (int64, error)
	Update(ctx context.Context, id int64, g goods_model.Goods, updatedBy int64) error
	SetPublished(ctx context.Context, id int64, published bool, updatedBy int64) error
	SetFeatured(ctx context.Context, id int64, featured bool, updatedBy int64) error
	Delete(ctx context.Context, id int64) error
	ReplaceImages(ctx context.Context, goodsID int64, urls []string) error
	ListImages(ctx context.Context, goodsID int64) ([]goods_model.Image, error)

	CategoryList(ctx context.Context, activeOnly bool) ([]goods_model.Category, error)
	CategoryFindByID(ctx context.Context, id int64) (goods_model.Category, error)
	CategorySlugExists(ctx context.Context, slug string, exceptID int64) (bool, error)
	CategoryCreate(ctx context.Context, cat goods_model.Category, createdBy int64) (int64, error)
	CategoryUpdate(ctx context.Context, id int64, cat goods_model.Category, updatedBy int64) error
	CategoryDelete(ctx context.Context, id int64) error
	CategoryInUse(ctx context.Context, id int64) (bool, error)
}
