package article_repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"fsldk-api/modules/article/article_model"

	"github.com/jmoiron/sqlx"
)

const selectCols = `a.articleID, a.articleTitle, a.articleSlug, a.articleExcerpt, a.articleContent, a.articleImage,
	a.categoryID, c.categoryName, a.isPublished, a.publishedDate, a.authorID, u.fullName AS authorName, a.createdDate`

const fromJoin = ` FROM ms_article a
	JOIN lk_article_category c ON c.categoryID = a.categoryID
	JOIN ms_user u ON u.userID = a.authorID`

// RepositoryImpl adalah implementasi Repository berbasis sqlx.
type RepositoryImpl struct{ db *sqlx.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *sqlx.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) List(ctx context.Context, f Filter) ([]article_model.Article, int, error) {
	where := " WHERE 1=1"
	args := []interface{}{}
	if f.PublishedOnly || f.Status == "published" {
		where += " AND a.isPublished = 1"
	} else if f.Status == "draft" {
		where += " AND a.isPublished = 0"
	}
	if f.Search != "" {
		where += " AND (a.articleTitle LIKE ? OR a.articleExcerpt LIKE ?)"
		like := "%" + f.Search + "%"
		args = append(args, like, like)
	}
	if f.CategorySlug != "" {
		where += " AND c.categorySlug = ?"
		args = append(args, f.CategorySlug)
	}
	if f.CategoryID > 0 {
		where += " AND a.categoryID = ?"
		args = append(args, f.CategoryID)
	}

	var total int
	if err := r.db.GetContext(ctx, &total, "SELECT COUNT(*)"+fromJoin+where, args...); err != nil {
		return nil, 0, err
	}
	q := "SELECT " + selectCols + fromJoin + where + " ORDER BY " + f.OrderBy + " LIMIT ? OFFSET ?"
	args = append(args, f.Limit, f.Offset)
	var out []article_model.Article
	if err := r.db.SelectContext(ctx, &out, q, args...); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *RepositoryImpl) findOne(ctx context.Context, where string, arg interface{}) (article_model.Article, error) {
	var a article_model.Article
	err := r.db.GetContext(ctx, &a, "SELECT "+selectCols+fromJoin+" WHERE "+where+" LIMIT 1", arg)
	if errors.Is(err, sql.ErrNoRows) {
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
	var n int
	err := r.db.GetContext(ctx, &n, "SELECT COUNT(*) FROM ms_article WHERE articleSlug = ? AND articleID <> ?", slug, exceptID)
	return n > 0, err
}

func (r *RepositoryImpl) Create(ctx context.Context, a article_model.Article, authorID int64) (int64, error) {
	var pub interface{}
	if a.IsPublished {
		pub = time.Now()
	}
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO ms_article (articleTitle, articleSlug, articleExcerpt, articleContent, articleImage,
			categoryID, isPublished, publishedDate, authorID, createdDate, createdBy)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), ?)`,
		a.ArticleTitle, a.ArticleSlug, a.ArticleExcerpt, a.ArticleContent, a.ArticleImage,
		a.CategoryID, a.IsPublished, pub, authorID, authorID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *RepositoryImpl) Update(ctx context.Context, id int64, a article_model.Article, updatedBy int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE ms_article SET articleTitle = ?, articleSlug = ?, articleExcerpt = ?, articleContent = ?,
			articleImage = ?, categoryID = ?, updatedDate = NOW(), updatedBy = ? WHERE articleID = ?`,
		a.ArticleTitle, a.ArticleSlug, a.ArticleExcerpt, a.ArticleContent, a.ArticleImage, a.CategoryID, updatedBy, id)
	return err
}

func (r *RepositoryImpl) SetPublished(ctx context.Context, id int64, published bool, updatedBy int64) error {
	if published {
		_, err := r.db.ExecContext(ctx,
			`UPDATE ms_article SET isPublished = 1, publishedDate = COALESCE(publishedDate, NOW()), updatedDate = NOW(), updatedBy = ? WHERE articleID = ?`,
			updatedBy, id)
		return err
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE ms_article SET isPublished = 0, updatedDate = NOW(), updatedBy = ? WHERE articleID = ?`, updatedBy, id)
	return err
}

func (r *RepositoryImpl) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM ms_article WHERE articleID = ?", id)
	return err
}

func (r *RepositoryImpl) Categories(ctx context.Context) ([]article_model.Category, error) {
	var out []article_model.Category
	err := r.db.SelectContext(ctx, &out,
		"SELECT categoryID, categoryName, categorySlug, isActive FROM lk_article_category WHERE isActive = 1 ORDER BY categoryName")
	return out, err
}
