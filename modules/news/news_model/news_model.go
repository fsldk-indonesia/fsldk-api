// Package news_model memuat entitas modul news.
package news_model

import "time"

// News merepresentasikan satu baris ms_news (dengan kolom join kategori & penulis).
type News struct {
	NewsID        int64      `db:"newsID" json:"newsID"`
	NewsTitle     string     `db:"newsTitle" json:"newsTitle"`
	NewsSlug      string     `db:"newsSlug" json:"newsSlug"`
	NewsExcerpt   *string    `db:"newsExcerpt" json:"newsExcerpt"`
	NewsContent   string     `db:"newsContent" json:"newsContent"`
	NewsImage     *string    `db:"newsImage" json:"newsImage"`
	CategoryID    int64      `db:"categoryID" json:"categoryID"`
	CategoryName  string     `db:"categoryName" json:"categoryName"`
	IsFeatured    bool       `db:"isFeatured" json:"isFeatured"`
	IsPublished   bool       `db:"isPublished" json:"isPublished"`
	PublishedDate *time.Time `db:"publishedDate" json:"publishedDate"`
	ViewCount     int64      `db:"viewCount" json:"viewCount"`
	AuthorID      int64      `db:"authorID" json:"authorID"`
	AuthorName    string     `db:"authorName" json:"authorName"`
	CreatedDate   time.Time  `db:"createdDate" json:"createdDate"`
}

// Category merepresentasikan satu baris lk_news_category.
type Category struct {
	CategoryID   int64  `db:"categoryID" json:"categoryID"`
	CategoryName string `db:"categoryName" json:"categoryName"`
	CategorySlug string `db:"categorySlug" json:"categorySlug"`
	IsActive     bool   `db:"isActive" json:"isActive"`
}
