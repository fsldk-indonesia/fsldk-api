// Package article mengelola artikel (publik & CMS) beserta kategorinya.
package article

import (
	"database/sql"
	"time"
)

// Article merepresentasikan satu baris ms_article.
type Article struct {
	ArticleID      int64          `db:"articleID" json:"articleID"`
	ArticleTitle   string         `db:"articleTitle" json:"articleTitle"`
	ArticleSlug    string         `db:"articleSlug" json:"articleSlug"`
	ArticleExcerpt sql.NullString `db:"articleExcerpt" json:"articleExcerpt"`
	ArticleContent string         `db:"articleContent" json:"articleContent"`
	ArticleImage   sql.NullString `db:"articleImage" json:"articleImage"`
	CategoryID     int64          `db:"categoryID" json:"categoryID"`
	CategoryName   string         `db:"categoryName" json:"categoryName"`
	IsPublished    bool           `db:"isPublished" json:"isPublished"`
	PublishedDate  sql.NullTime   `db:"publishedDate" json:"publishedDate"`
	AuthorID       int64          `db:"authorID" json:"authorID"`
	AuthorName     string         `db:"authorName" json:"authorName"`
	CreatedDate    time.Time      `db:"createdDate" json:"createdDate"`
}

// Category merepresentasikan satu baris lk_article_category.
type Category struct {
	CategoryID   int64  `db:"categoryID" json:"categoryID"`
	CategoryName string `db:"categoryName" json:"categoryName"`
	CategorySlug string `db:"categorySlug" json:"categorySlug"`
	IsActive     bool   `db:"isActive" json:"isActive"`
}

// Request adalah body membuat/memperbarui artikel.
type Request struct {
	ArticleTitle   string `json:"articleTitle" validate:"required,min=3,max=255"`
	ArticleExcerpt string `json:"articleExcerpt" validate:"max=500"`
	ArticleContent string `json:"articleContent" validate:"required"`
	ArticleImage   string `json:"articleImage" validate:"max=255"`
	CategoryID     int64  `json:"categoryID" validate:"required"`
	Status         string `json:"status" validate:"omitempty,oneof=draft published"`
}
