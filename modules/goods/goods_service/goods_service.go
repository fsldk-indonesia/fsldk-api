// Package goods_service memuat logika bisnis modul goods.
package goods_service

import (
	"context"

	"fsldk-api/base/dto"
	"fsldk-api/modules/goods/goods_dto"
	"fsldk-api/modules/goods/goods_model"
)

// Service adalah kontrak logika bisnis produk & kategori goods.
type Service interface {
	PublicList(ctx context.Context, q dto.ListQuery, f goods_dto.Filter, sort string) ([]goods_model.Goods, int, error)
	CMSList(ctx context.Context, q dto.ListQuery, f goods_dto.Filter) ([]goods_model.Goods, int, error)
	PublicDetail(ctx context.Context, slug string) (goods_dto.DetailResponse, error)
	Get(ctx context.Context, id int64) (goods_dto.DetailResponse, error)
	Create(ctx context.Context, req goods_dto.Request, actorID int64) (goods_dto.DetailResponse, error)
	Update(ctx context.Context, id int64, req goods_dto.Request, actorID int64) (goods_dto.DetailResponse, error)
	SetPublished(ctx context.Context, id int64, published bool, actorID int64) error
	SetFeatured(ctx context.Context, id int64, featured bool, actorID int64) error
	Delete(ctx context.Context, id int64) error

	PublicCategories(ctx context.Context) ([]goods_model.Category, error)
	CMSCategories(ctx context.Context) ([]goods_model.Category, error)
	CategoryCreate(ctx context.Context, req goods_dto.CategoryRequest, actorID int64) (goods_model.Category, error)
	CategoryUpdate(ctx context.Context, id int64, req goods_dto.CategoryRequest, actorID int64) (goods_model.Category, error)
	CategoryDelete(ctx context.Context, id int64) error
}
