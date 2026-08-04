package article_repository

import (
	"context"
	"errors"
	"time"

	"fsldk-api/modules/article/article_dto"
	"fsldk-api/modules/article/article_model"

	"gorm.io/gorm"
)

const selectCols = "a.articleID, a.articleTitle, a.articleSlug, a.articleIntro, a.articleImage, " +
	"a.articleWriter, a.articleEditor, a.articlePdf, " +
	"a.categoryID, c.categoryName, a.isPublished, a.publishedDate, a.authorID, u.fullName AS authorName, a.createdDate"

// RepositoryImpl adalah implementasi Repository berbasis GORM.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) baseQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Table("ms_article a").
		Joins("JOIN lk_article_category c ON c.categoryID = a.categoryID").
		Joins("JOIN ms_user u ON u.userID = a.authorID")
}

func (r *RepositoryImpl) List(ctx context.Context, f article_dto.Filter) ([]article_model.Article, int64, error) {
	q := r.baseQuery(ctx)
	if f.PublishedOnly || f.Status == "published" {
		q = q.Where("a.isPublished = 1")
	} else if f.Status == "draft" {
		q = q.Where("a.isPublished = 0")
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("(a.articleTitle LIKE ? OR a.articleWriter LIKE ?)", like, like)
	}
	if f.CategorySlug != "" {
		q = q.Where("c.categorySlug = ?", f.CategorySlug)
	}
	if f.CategoryID > 0 {
		q = q.Where("a.categoryID = ?", f.CategoryID)
	}

	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []article_model.Article
	err := q.Select(selectCols).Order(f.OrderBy).Limit(f.Limit).Offset(f.Offset).Find(&out).Error
	return out, total, err
}

func (r *RepositoryImpl) findOne(ctx context.Context, where string, arg interface{}) (article_model.Article, error) {
	var a article_model.Article
	err := r.baseQuery(ctx).Select(selectCols).Where(where, arg).Take(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return article_model.Article{}, ErrNotFound
	}
	return a, err
}

func (r *RepositoryImpl) FindByID(ctx context.Context, id int64) (article_model.Article, error) {
	return r.findOne(ctx, "a.articleID = ?", id)
}

func (r *RepositoryImpl) FindBySlug(ctx context.Context, slug string) (article_model.Article, error) {
	return r.findOne(ctx, "a.articleSlug = ?", slug)
}

func (r *RepositoryImpl) SlugExists(ctx context.Context, slug string, exceptID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("ms_article").
		Where("articleSlug = ? AND articleID <> ?", slug, exceptID).Count(&count).Error
	return count > 0, err
}

func (r *RepositoryImpl) Create(ctx context.Context, a article_model.Article, authorID int64) (int64, error) {
	var publishedAt interface{}
	if a.IsPublished {
		publishedAt = time.Now()
	}
	values := map[string]interface{}{
		"articleTitle":  a.ArticleTitle,
		"articleSlug":   a.ArticleSlug,
		"articleIntro":  a.ArticleIntro,
		"articleImage":  a.ArticleImage,
		"articleWriter": a.ArticleWriter,
		"articleEditor": a.ArticleEditor,
		"articlePdf":    a.ArticlePdf,
		"categoryID":    a.CategoryID,
		"isPublished":   a.IsPublished,
		"publishedDate": publishedAt,
		"authorID":      authorID,
		"createdDate":   time.Now(),
		"createdBy":     authorID,
	}
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("ms_article").Create(values).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) Update(ctx context.Context, id int64, a article_model.Article, updatedBy int64) error {
	return r.db.WithContext(ctx).Table("ms_article").Where("articleID = ?", id).Updates(map[string]interface{}{
		"articleTitle":  a.ArticleTitle,
		"articleSlug":   a.ArticleSlug,
		"articleIntro":  a.ArticleIntro,
		"articleImage":  a.ArticleImage,
		"articleWriter": a.ArticleWriter,
		"articleEditor": a.ArticleEditor,
		"articlePdf":    a.ArticlePdf,
		"categoryID":    a.CategoryID,
		"updatedDate":   time.Now(),
		"updatedBy":     updatedBy,
	}).Error
}

func (r *RepositoryImpl) SetPublished(ctx context.Context, id int64, published bool, updatedBy int64) error {
	if published {
		return r.db.WithContext(ctx).Exec(
			"UPDATE ms_article SET isPublished = 1, publishedDate = COALESCE(publishedDate, NOW()), updatedDate = NOW(), updatedBy = ? WHERE articleID = ?",
			updatedBy, id).Error
	}
	return r.db.WithContext(ctx).Table("ms_article").Where("articleID = ?", id).Updates(map[string]interface{}{
		"isPublished": false,
		"updatedDate": time.Now(),
		"updatedBy":   updatedBy,
	}).Error
}

func (r *RepositoryImpl) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Exec("DELETE FROM ms_article WHERE articleID = ?", id).Error
}

func (r *RepositoryImpl) Categories(ctx context.Context) ([]article_model.Category, error) {
	var out []article_model.Category
	err := r.db.WithContext(ctx).Table("lk_article_category").
		Where("isActive = 1").Order("categoryName").Find(&out).Error
	return out, err
}
