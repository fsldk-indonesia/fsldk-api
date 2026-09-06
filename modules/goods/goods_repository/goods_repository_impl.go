package goods_repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"fsldk-api/constants"
	"fsldk-api/modules/goods/goods_dto"
	"fsldk-api/modules/goods/goods_model"
)

const selectCols = "g.goodsID, g.goodsName, g.goodsSlug, g.skuCode, g.goodsCategoryID, c.categoryName, " +
	"g.shortDescription, g.fullDescription, g.price, g.mainImageUrl, g.availabilityStatus, " +
	"g.isFeatured, g.isPublished, g.publishedDate, g.sortOrder, g.purchaseUrl, g.purchaseButtonLabel, g.createdDate"

// RepositoryImpl adalah implementasi Repository berbasis GORM.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) baseQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Table(constants.TableGoods + " g").
		Joins("JOIN " + constants.TableGoodsCategory + " c ON c.goodsCategoryID = g.goodsCategoryID")
}

func (r *RepositoryImpl) List(ctx context.Context, f goods_dto.Filter) ([]goods_model.Goods, int64, error) {
	q := r.baseQuery(ctx)
	if f.PublishedOnly {
		q = q.Where("g.isPublished = 1")
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("(g.goodsName LIKE ? OR g.skuCode LIKE ?)", like, like)
	}
	if f.CategorySlug != "" {
		q = q.Where("c.categorySlug = ?", f.CategorySlug)
	}
	if f.CategoryID > 0 {
		q = q.Where("g.goodsCategoryID = ?", f.CategoryID)
	}
	if f.Availability != "" {
		q = q.Where("g.availabilityStatus = ?", f.Availability)
	}
	if f.FeaturedOnly {
		q = q.Where("g.isFeatured = 1")
	}

	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []goods_model.Goods
	err := q.Select(selectCols).Order(f.OrderBy).Limit(f.Limit).Offset(f.Offset).Find(&out).Error
	return out, total, err
}

func (r *RepositoryImpl) findOne(ctx context.Context, where string, arg interface{}) (goods_model.Goods, error) {
	var g goods_model.Goods
	err := r.baseQuery(ctx).Select(selectCols).Where(where, arg).Take(&g).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return goods_model.Goods{}, ErrNotFound
	}
	return g, err
}

func (r *RepositoryImpl) FindByID(ctx context.Context, id int64) (goods_model.Goods, error) {
	return r.findOne(ctx, "g.goodsID = ?", id)
}

func (r *RepositoryImpl) FindBySlug(ctx context.Context, slug string) (goods_model.Goods, error) {
	return r.findOne(ctx, "g.goodsSlug = ?", slug)
}

func (r *RepositoryImpl) SlugExists(ctx context.Context, slug string, exceptID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table(constants.TableGoods).
		Where("goodsSlug = ? AND goodsID <> ?", slug, exceptID).Count(&count).Error
	return count > 0, err
}

func (r *RepositoryImpl) SKUExists(ctx context.Context, sku string, exceptID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table(constants.TableGoods).
		Where("skuCode = ? AND goodsID <> ?", sku, exceptID).Count(&count).Error
	return count > 0, err
}

func (r *RepositoryImpl) Create(ctx context.Context, g goods_model.Goods, createdBy int64) (int64, error) {
	var publishedAt interface{}
	if g.IsPublished {
		publishedAt = time.Now()
	}
	values := map[string]interface{}{
		"goodsName":           g.GoodsName,
		"goodsSlug":           g.GoodsSlug,
		"skuCode":             g.SKUCode,
		"goodsCategoryID":     g.GoodsCategoryID,
		"shortDescription":    g.ShortDescription,
		"fullDescription":     g.FullDescription,
		"price":               g.Price,
		"mainImageUrl":        g.MainImageUrl,
		"availabilityStatus":  g.AvailabilityStatus,
		"isFeatured":          g.IsFeatured,
		"isPublished":         g.IsPublished,
		"publishedDate":       publishedAt,
		"purchaseUrl":         g.PurchaseUrl,
		"purchaseButtonLabel": g.PurchaseButtonLabel,
		"createdDate":         time.Now(),
		"createdBy":           createdBy,
	}
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table(constants.TableGoods).Create(values).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) Update(ctx context.Context, id int64, g goods_model.Goods, updatedBy int64) error {
	return r.db.WithContext(ctx).Table(constants.TableGoods).Where("goodsID = ?", id).Updates(map[string]interface{}{
		"goodsName":           g.GoodsName,
		"goodsSlug":           g.GoodsSlug,
		"skuCode":             g.SKUCode,
		"goodsCategoryID":     g.GoodsCategoryID,
		"shortDescription":    g.ShortDescription,
		"fullDescription":     g.FullDescription,
		"price":               g.Price,
		"mainImageUrl":        g.MainImageUrl,
		"availabilityStatus":  g.AvailabilityStatus,
		"purchaseUrl":         g.PurchaseUrl,
		"purchaseButtonLabel": g.PurchaseButtonLabel,
		"updatedDate":         time.Now(),
		"updatedBy":           updatedBy,
	}).Error
}

func (r *RepositoryImpl) SetPublished(ctx context.Context, id int64, published bool, updatedBy int64) error {
	if published {
		return r.db.WithContext(ctx).Exec(
			"UPDATE "+constants.TableGoods+" SET isPublished = 1, publishedDate = COALESCE(publishedDate, NOW()), updatedDate = NOW(), updatedBy = ? WHERE goodsID = ?",
			updatedBy, id).Error
	}
	return r.db.WithContext(ctx).Table(constants.TableGoods).Where("goodsID = ?", id).Updates(map[string]interface{}{
		"isPublished": false,
		"updatedDate": time.Now(),
		"updatedBy":   updatedBy,
	}).Error
}

func (r *RepositoryImpl) SetFeatured(ctx context.Context, id int64, featured bool, updatedBy int64) error {
	return r.db.WithContext(ctx).Table(constants.TableGoods).Where("goodsID = ?", id).Updates(map[string]interface{}{
		"isFeatured":  featured,
		"updatedDate": time.Now(),
		"updatedBy":   updatedBy,
	}).Error
}

func (r *RepositoryImpl) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Exec("DELETE FROM "+constants.TableGoods+" WHERE goodsID = ?", id).Error
}

func (r *RepositoryImpl) ReplaceImages(ctx context.Context, goodsID int64, urls []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM "+constants.TableGoodsImage+" WHERE goodsID = ?", goodsID).Error; err != nil {
			return err
		}
		for i, u := range urls {
			if err := tx.Table(constants.TableGoodsImage).Create(map[string]interface{}{
				"goodsID":   goodsID,
				"imageUrl":  u,
				"sortOrder": i,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *RepositoryImpl) ListImages(ctx context.Context, goodsID int64) ([]goods_model.Image, error) {
	var out []goods_model.Image
	err := r.db.WithContext(ctx).Table(constants.TableGoodsImage).
		Where("goodsID = ?", goodsID).Order("sortOrder").Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) CategoryList(ctx context.Context, activeOnly bool) ([]goods_model.Category, error) {
	q := r.db.WithContext(ctx).Table(constants.TableGoodsCategory)
	if activeOnly {
		q = q.Where("isActive = 1")
	}
	var out []goods_model.Category
	err := q.Order("sortOrder, categoryName").Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) CategoryFindByID(ctx context.Context, id int64) (goods_model.Category, error) {
	var cat goods_model.Category
	err := r.db.WithContext(ctx).Table(constants.TableGoodsCategory).Where("goodsCategoryID = ?", id).Take(&cat).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return goods_model.Category{}, ErrCategoryNotFound
	}
	return cat, err
}

func (r *RepositoryImpl) CategorySlugExists(ctx context.Context, slug string, exceptID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table(constants.TableGoodsCategory).
		Where("categorySlug = ? AND goodsCategoryID <> ?", slug, exceptID).Count(&count).Error
	return count > 0, err
}

func (r *RepositoryImpl) CategoryCreate(ctx context.Context, cat goods_model.Category, createdBy int64) (int64, error) {
	values := map[string]interface{}{
		"categoryName": cat.CategoryName,
		"categorySlug": cat.CategorySlug,
		"isActive":     cat.IsActive,
		"sortOrder":    cat.SortOrder,
		"createdDate":  time.Now(),
		"createdBy":    createdBy,
	}
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table(constants.TableGoodsCategory).Create(values).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) CategoryUpdate(ctx context.Context, id int64, cat goods_model.Category, updatedBy int64) error {
	return r.db.WithContext(ctx).Table(constants.TableGoodsCategory).Where("goodsCategoryID = ?", id).Updates(map[string]interface{}{
		"categoryName": cat.CategoryName,
		"categorySlug": cat.CategorySlug,
		"isActive":     cat.IsActive,
		"sortOrder":    cat.SortOrder,
		"updatedDate":  time.Now(),
		"updatedBy":    updatedBy,
	}).Error
}

func (r *RepositoryImpl) CategoryDelete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Exec("DELETE FROM "+constants.TableGoodsCategory+" WHERE goodsCategoryID = ?", id).Error
}

func (r *RepositoryImpl) CategoryInUse(ctx context.Context, id int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table(constants.TableGoods).Where("goodsCategoryID = ?", id).Count(&count).Error
	return count > 0, err
}
