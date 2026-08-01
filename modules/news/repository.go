package news

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
)

// ErrNotFound dikembalikan bila berita tidak ditemukan.
var ErrNotFound = errors.New("berita tidak ditemukan")

const selectCols = `n.newsID, n.newsTitle, n.newsSlug, n.newsExcerpt, n.newsContent, n.newsImage,
	n.categoryID, c.categoryName, n.isFeatured, n.isPublished, n.publishedDate, n.viewCount,
	n.authorID, u.fullName AS authorName, n.createdDate`

const fromJoin = ` FROM ms_news n
	JOIN lk_news_category c ON c.categoryID = n.categoryID
	JOIN ms_user u ON u.userID = n.authorID`

// Filter menampung parameter penyaringan daftar berita.
type Filter struct {
	Search        string
	CategorySlug  string
	CategoryID    int64
	PublishedOnly bool
	Status        string // "published" | "draft" | ""
	Limit         int
	Offset        int
	OrderBy       string
}

// Repository adalah kontrak akses data berita.
type Repository interface {
	List(ctx context.Context, f Filter) ([]News, int, error)
	FindByID(ctx context.Context, id int64) (News, error)
	FindBySlug(ctx context.Context, slug string) (News, error)
	Featured(ctx context.Context, limit int) ([]News, error)
	SlugExists(ctx context.Context, slug string, exceptID int64) (bool, error)
	Create(ctx context.Context, n News, authorID int64) (int64, error)
	Update(ctx context.Context, id int64, n News, updatedBy int64) error
	SetPublished(ctx context.Context, id int64, published bool, updatedBy int64) error
	SetFeatured(ctx context.Context, id int64, featured bool, updatedBy int64) error
	Delete(ctx context.Context, id int64) error
	IncrementView(ctx context.Context, id int64) error
	Categories(ctx context.Context) ([]Category, error)
	CountByPublished(ctx context.Context, published bool) (int, error)
}

type repository struct{ db *sqlx.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *sqlx.DB) Repository { return &repository{db: db} }

func (r *repository) List(ctx context.Context, f Filter) ([]News, int, error) {
	where := " WHERE 1=1"
	args := []interface{}{}
	if f.PublishedOnly {
		where += " AND n.isPublished = 1"
	}
	if f.Status == "published" {
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
	var out []News
	if err := r.db.SelectContext(ctx, &out, q, args...); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *repository) findOne(ctx context.Context, where string, arg interface{}) (News, error) {
	var n News
	err := r.db.GetContext(ctx, &n, "SELECT "+selectCols+fromJoin+" WHERE "+where+" LIMIT 1", arg)
	if errors.Is(err, sql.ErrNoRows) {
		return News{}, ErrNotFound
	}
	return n, err
}

func (r *repository) FindByID(ctx context.Context, id int64) (News, error) {
	return r.findOne(ctx, "n.newsID = ?", id)
}

func (r *repository) FindBySlug(ctx context.Context, slug string) (News, error) {
	return r.findOne(ctx, "n.newsSlug = ?", slug)
}

func (r *repository) Featured(ctx context.Context, limit int) ([]News, error) {
	var out []News
	err := r.db.SelectContext(ctx, &out,
		"SELECT "+selectCols+fromJoin+" WHERE n.isPublished = 1 AND n.isFeatured = 1 ORDER BY n.publishedDate DESC LIMIT ?", limit)
	return out, err
}

func (r *repository) SlugExists(ctx context.Context, slug string, exceptID int64) (bool, error) {
	var n int
	err := r.db.GetContext(ctx, &n, "SELECT COUNT(*) FROM ms_news WHERE newsSlug = ? AND newsID <> ?", slug, exceptID)
	return n > 0, err
}

func (r *repository) Create(ctx context.Context, n News, authorID int64) (int64, error) {
	var pub interface{}
	if n.IsPublished {
		pub = sql.NullTime{Time: time.Now(), Valid: true}
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

func (r *repository) Update(ctx context.Context, id int64, n News, updatedBy int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE ms_news SET newsTitle = ?, newsSlug = ?, newsExcerpt = ?, newsContent = ?, newsImage = ?,
			categoryID = ?, isFeatured = ?, updatedDate = NOW(), updatedBy = ? WHERE newsID = ?`,
		n.NewsTitle, n.NewsSlug, n.NewsExcerpt, n.NewsContent, n.NewsImage, n.CategoryID, n.IsFeatured, updatedBy, id)
	return err
}

func (r *repository) SetPublished(ctx context.Context, id int64, published bool, updatedBy int64) error {
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

func (r *repository) SetFeatured(ctx context.Context, id int64, featured bool, updatedBy int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE ms_news SET isFeatured = ?, updatedDate = NOW(), updatedBy = ? WHERE newsID = ?`, featured, updatedBy, id)
	return err
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM ms_news WHERE newsID = ?", id)
	return err
}

func (r *repository) IncrementView(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "UPDATE ms_news SET viewCount = viewCount + 1 WHERE newsID = ?", id)
	return err
}

func (r *repository) Categories(ctx context.Context) ([]Category, error) {
	var out []Category
	err := r.db.SelectContext(ctx, &out,
		"SELECT categoryID, categoryName, categorySlug, isActive FROM lk_news_category WHERE isActive = 1 ORDER BY categoryName")
	return out, err
}

func (r *repository) CountByPublished(ctx context.Context, published bool) (int, error) {
	var n int
	err := r.db.GetContext(ctx, &n, "SELECT COUNT(*) FROM ms_news WHERE isPublished = ?", published)
	return n, err
}
