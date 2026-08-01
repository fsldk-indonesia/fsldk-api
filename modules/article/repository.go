package article

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
)

// ErrNotFound dikembalikan bila artikel tidak ditemukan.
var ErrNotFound = errors.New("artikel tidak ditemukan")

const selectCols = `a.articleID, a.articleTitle, a.articleSlug, a.articleExcerpt, a.articleContent, a.articleImage,
	a.categoryID, c.categoryName, a.isPublished, a.publishedDate, a.authorID, u.fullName AS authorName, a.createdDate`

const fromJoin = ` FROM ms_article a
	JOIN lk_article_category c ON c.categoryID = a.categoryID
	JOIN ms_user u ON u.userID = a.authorID`

// Filter menampung parameter penyaringan daftar artikel.
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

// Repository adalah kontrak akses data artikel.
type Repository interface {
	List(ctx context.Context, f Filter) ([]Article, int, error)
	FindByID(ctx context.Context, id int64) (Article, error)
	FindBySlug(ctx context.Context, slug string) (Article, error)
	SlugExists(ctx context.Context, slug string, exceptID int64) (bool, error)
	Create(ctx context.Context, a Article, authorID int64) (int64, error)
	Update(ctx context.Context, id int64, a Article, updatedBy int64) error
	SetPublished(ctx context.Context, id int64, published bool, updatedBy int64) error
	Delete(ctx context.Context, id int64) error
	Categories(ctx context.Context) ([]Category, error)
}

type repository struct{ db *sqlx.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *sqlx.DB) Repository { return &repository{db: db} }

func (r *repository) List(ctx context.Context, f Filter) ([]Article, int, error) {
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
	var out []Article
	if err := r.db.SelectContext(ctx, &out, q, args...); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *repository) findOne(ctx context.Context, where string, arg interface{}) (Article, error) {
	var a Article
	err := r.db.GetContext(ctx, &a, "SELECT "+selectCols+fromJoin+" WHERE "+where+" LIMIT 1", arg)
	if errors.Is(err, sql.ErrNoRows) {
		return Article{}, ErrNotFound
	}
	return a, err
}

func (r *repository) FindByID(ctx context.Context, id int64) (Article, error) {
	return r.findOne(ctx, "a.articleID = ?", id)
}

func (r *repository) FindBySlug(ctx context.Context, slug string) (Article, error) {
	return r.findOne(ctx, "a.articleSlug = ?", slug)
}

func (r *repository) SlugExists(ctx context.Context, slug string, exceptID int64) (bool, error) {
	var n int
	err := r.db.GetContext(ctx, &n, "SELECT COUNT(*) FROM ms_article WHERE articleSlug = ? AND articleID <> ?", slug, exceptID)
	return n > 0, err
}

func (r *repository) Create(ctx context.Context, a Article, authorID int64) (int64, error) {
	var pub interface{}
	if a.IsPublished {
		pub = sql.NullTime{Time: time.Now(), Valid: true}
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

func (r *repository) Update(ctx context.Context, id int64, a Article, updatedBy int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE ms_article SET articleTitle = ?, articleSlug = ?, articleExcerpt = ?, articleContent = ?,
			articleImage = ?, categoryID = ?, updatedDate = NOW(), updatedBy = ? WHERE articleID = ?`,
		a.ArticleTitle, a.ArticleSlug, a.ArticleExcerpt, a.ArticleContent, a.ArticleImage, a.CategoryID, updatedBy, id)
	return err
}

func (r *repository) SetPublished(ctx context.Context, id int64, published bool, updatedBy int64) error {
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

func (r *repository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM ms_article WHERE articleID = ?", id)
	return err
}

func (r *repository) Categories(ctx context.Context) ([]Category, error) {
	var out []Category
	err := r.db.SelectContext(ctx, &out,
		"SELECT categoryID, categoryName, categorySlug, isActive FROM lk_article_category WHERE isActive = 1 ORDER BY categoryName")
	return out, err
}
