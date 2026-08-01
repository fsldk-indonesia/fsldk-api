package news_repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"fsldk-api/modules/news/news_model"

	"github.com/jmoiron/sqlx"
)

const selectCols = `n.newsID, n.newsTitle, n.newsSlug, n.newsExcerpt, n.newsContent, n.newsImage,
	n.categoryID, c.categoryName, n.isFeatured, n.isPublished, n.publishedDate, n.viewCount,
	n.authorID, u.fullName AS authorName, n.createdDate`

const fromJoin = ` FROM ms_news n
	JOIN lk_news_category c ON c.categoryID = n.categoryID
	JOIN ms_user u ON u.userID = n.authorID`

// RepositoryImpl adalah implementasi Repository berbasis sqlx.
type RepositoryImpl struct{ db *sqlx.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *sqlx.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) List(ctx context.Context, f Filter) ([]news_model.News, int, error) {
	where := " WHERE 1=1"
	args := []interface{}{}
	if f.PublishedOnly || f.Status == "published" {
		where += " AND n.isPublished = 1"
	} else if f.Status == "draft" {
		where += " AND n.isPublished = 0"
	}
	if f.Search != "" {
		where += " AND (n.newsTitle LIKE ? OR n.newsExcerpt LIKE ?)"
		like := "%" + f.Search + "%"
		args = append(args, like, like)
	}
	if f.CategorySlug != "" {
		where += " AND c.categorySlug = ?"
		args = append(args, f.CategorySlug)
	}
	if f.CategoryID > 0 {
		where += " AND n.categoryID = ?"
		args = append(args, f.CategoryID)
	}

	var total int
	if err := r.db.GetContext(ctx, &total, "SELECT COUNT(*)"+fromJoin+where, args...); err != nil {
		return nil, 0, err
	}

	q := "SELECT " + selectCols + fromJoin + where + " ORDER BY " + f.OrderBy + " LIMIT ? OFFSET ?"
	args = append(args, f.Limit, f.Offset)
	var out []news_model.News
	if err := r.db.SelectContext(ctx, &out, q, args...); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *RepositoryImpl) findOne(ctx context.Context, where string, arg interface{}) (news_model.News, error) {
	var n news_model.News
	err := r.db.GetContext(ctx, &n, "SELECT "+selectCols+fromJoin+" WHERE "+where+" LIMIT 1", arg)
	if errors.Is(err, sql.ErrNoRows) {
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
	err := r.db.SelectContext(ctx, &out,
		"SELECT "+selectCols+fromJoin+" WHERE n.isPublished = 1 AND n.isFeatured = 1 ORDER BY n.publishedDate DESC LIMIT ?", limit)
	return out, err
}

func (r *RepositoryImpl) SlugExists(ctx context.Context, slug string, exceptID int64) (bool, error) {
	var n int
	err := r.db.GetContext(ctx, &n, "SELECT COUNT(*) FROM ms_news WHERE newsSlug = ? AND newsID <> ?", slug, exceptID)
	return n > 0, err
}

func (r *RepositoryImpl) Create(ctx context.Context, n news_model.News, authorID int64) (int64, error) {
	var pub interface{}
	if n.IsPublished {
		pub = time.Now()
	}
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO ms_news (newsTitle, newsSlug, newsExcerpt, newsContent, newsImage, categoryID,
			isFeatured, isPublished, publishedDate, authorID, createdDate, createdBy)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), ?)`,
		n.NewsTitle, n.NewsSlug, n.NewsExcerpt, n.NewsContent, n.NewsImage, n.CategoryID,
		n.IsFeatured, n.IsPublished, pub, authorID, authorID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *RepositoryImpl) Update(ctx context.Context, id int64, n news_model.News, updatedBy int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE ms_news SET newsTitle = ?, newsSlug = ?, newsExcerpt = ?, newsContent = ?, newsImage = ?,
			categoryID = ?, isFeatured = ?, updatedDate = NOW(), updatedBy = ? WHERE newsID = ?`,
		n.NewsTitle, n.NewsSlug, n.NewsExcerpt, n.NewsContent, n.NewsImage, n.CategoryID, n.IsFeatured, updatedBy, id)
	return err
}

func (r *RepositoryImpl) SetPublished(ctx context.Context, id int64, published bool, updatedBy int64) error {
	if published {
		_, err := r.db.ExecContext(ctx,
			`UPDATE ms_news SET isPublished = 1, publishedDate = COALESCE(publishedDate, NOW()), updatedDate = NOW(), updatedBy = ? WHERE newsID = ?`,
			updatedBy, id)
		return err
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE ms_news SET isPublished = 0, updatedDate = NOW(), updatedBy = ? WHERE newsID = ?`, updatedBy, id)
	return err
}

func (r *RepositoryImpl) SetFeatured(ctx context.Context, id int64, featured bool, updatedBy int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE ms_news SET isFeatured = ?, updatedDate = NOW(), updatedBy = ? WHERE newsID = ?`, featured, updatedBy, id)
	return err
}

func (r *RepositoryImpl) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM ms_news WHERE newsID = ?", id)
	return err
}

func (r *RepositoryImpl) IncrementView(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "UPDATE ms_news SET viewCount = viewCount + 1 WHERE newsID = ?", id)
	return err
}

func (r *RepositoryImpl) Categories(ctx context.Context) ([]news_model.Category, error) {
	var out []news_model.Category
	err := r.db.SelectContext(ctx, &out,
		"SELECT categoryID, categoryName, categorySlug, isActive FROM lk_news_category WHERE isActive = 1 ORDER BY categoryName")
	return out, err
}
