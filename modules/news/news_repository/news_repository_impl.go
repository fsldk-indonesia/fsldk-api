package news_repository

import (
	"context"
	"errors"
	"time"

	"fsldk-api/modules/news/news_dto"
	"fsldk-api/modules/news/news_model"

	"gorm.io/gorm"
)

const selectCols = "n.newsID, n.newsTitle, n.newsSlug, n.newsExcerpt, n.newsContent, n.newsImage, " +
	"n.categoryID, c.categoryName, n.isFeatured, n.isPublished, n.publishedDate, n.viewCount, " +
	"n.authorID, u.fullName AS authorName, n.createdDate"

// RepositoryImpl adalah implementasi Repository berbasis GORM.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) baseQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Table("ms_news n").
		Joins("JOIN lk_news_category c ON c.categoryID = n.categoryID").
		Joins("JOIN ms_user u ON u.userID = n.authorID")
}

func (r *RepositoryImpl) List(ctx context.Context, f news_dto.Filter) ([]news_model.News, int64, error) {
	q := r.baseQuery(ctx)
	if f.PublishedOnly || f.Status == "published" {
		q = q.Where("n.isPublished = 1")
	} else if f.Status == "draft" {
		q = q.Where("n.isPublished = 0")
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("(n.newsTitle LIKE ? OR n.newsExcerpt LIKE ?)", like, like)
	}
	if f.CategorySlug != "" {
		q = q.Where("c.categorySlug = ?", f.CategorySlug)
	}
	if f.CategoryID > 0 {
		q = q.Where("n.categoryID = ?", f.CategoryID)
	}

	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []news_model.News
	err := q.Select(selectCols).Order(f.OrderBy).Limit(f.Limit).Offset(f.Offset).Find(&out).Error
	return out, total, err
}

func (r *RepositoryImpl) findOne(ctx context.Context, where string, arg interface{}) (news_model.News, error) {
	var n news_model.News
	err := r.baseQuery(ctx).Select(selectCols).Where(where, arg).Take(&n).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return news_model.News{}, ErrNotFound
	}
	return n, err
}

func (r *RepositoryImpl) FindByID(ctx context.Context, id int64) (news_model.News, error) {
	return r.findOne(ctx, "n.newsID = ?", id)
}

func (r *RepositoryImpl) FindBySlug(ctx context.Context, slug string) (news_model.News, error) {
	return r.findOne(ctx, "n.newsSlug = ?", slug)
}

func (r *RepositoryImpl) Featured(ctx context.Context, limit int) ([]news_model.News, error) {
	var out []news_model.News
	err := r.baseQuery(ctx).Select(selectCols).
		Where("n.isPublished = 1 AND n.isFeatured = 1").
		Order("n.publishedDate DESC").Limit(limit).Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) SlugExists(ctx context.Context, slug string, exceptID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("ms_news").
		Where("newsSlug = ? AND newsID <> ?", slug, exceptID).Count(&count).Error
	return count > 0, err
}

func (r *RepositoryImpl) Create(ctx context.Context, n news_model.News, authorID int64) (int64, error) {
	var publishedAt interface{}
	if n.IsPublished {
		publishedAt = time.Now()
	}
	values := map[string]interface{}{
		"newsTitle":     n.NewsTitle,
		"newsSlug":      n.NewsSlug,
		"newsExcerpt":   n.NewsExcerpt,
		"newsContent":   n.NewsContent,
		"newsImage":     n.NewsImage,
		"categoryID":    n.CategoryID,
		"isFeatured":    n.IsFeatured,
		"isPublished":   n.IsPublished,
		"publishedDate": publishedAt,
		"authorID":      authorID,
		"createdDate":   time.Now(),
		"createdBy":     authorID,
	}
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("ms_news").Create(values).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) Update(ctx context.Context, id int64, n news_model.News, updatedBy int64) error {
	return r.db.WithContext(ctx).Table("ms_news").Where("newsID = ?", id).Updates(map[string]interface{}{
		"newsTitle":   n.NewsTitle,
		"newsSlug":    n.NewsSlug,
		"newsExcerpt": n.NewsExcerpt,
		"newsContent": n.NewsContent,
		"newsImage":   n.NewsImage,
		"categoryID":  n.CategoryID,
		"isFeatured":  n.IsFeatured,
		"updatedDate": time.Now(),
		"updatedBy":   updatedBy,
	}).Error
}

func (r *RepositoryImpl) SetPublished(ctx context.Context, id int64, published bool, updatedBy int64) error {
	if published {
		return r.db.WithContext(ctx).Exec(
			"UPDATE ms_news SET isPublished = 1, publishedDate = COALESCE(publishedDate, NOW()), updatedDate = NOW(), updatedBy = ? WHERE newsID = ?",
			updatedBy, id).Error
	}
	return r.db.WithContext(ctx).Table("ms_news").Where("newsID = ?", id).Updates(map[string]interface{}{
		"isPublished": false,
		"updatedDate": time.Now(),
		"updatedBy":   updatedBy,
	}).Error
}

func (r *RepositoryImpl) SetFeatured(ctx context.Context, id int64, featured bool, updatedBy int64) error {
	return r.db.WithContext(ctx).Table("ms_news").Where("newsID = ?", id).Updates(map[string]interface{}{
		"isFeatured":  featured,
		"updatedDate": time.Now(),
		"updatedBy":   updatedBy,
	}).Error
}

func (r *RepositoryImpl) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Exec("DELETE FROM ms_news WHERE newsID = ?", id).Error
}

func (r *RepositoryImpl) IncrementView(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Exec("UPDATE ms_news SET viewCount = viewCount + 1 WHERE newsID = ?", id).Error
}

func (r *RepositoryImpl) Categories(ctx context.Context) ([]news_model.Category, error) {
	var out []news_model.Category
	err := r.db.WithContext(ctx).Table("lk_news_category").
		Where("isActive = 1").Order("categoryName").Find(&out).Error
	return out, err
}
