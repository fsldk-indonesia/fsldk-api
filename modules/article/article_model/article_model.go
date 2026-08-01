// Package article_model memuat entitas modul article.
package article_model

import "time"

// Article merepresentasikan satu baris ms_article.
type Article struct {
	ArticleID      int64      `db:"articleID" json:"articleID"`
	ArticleTitle   string     `db:"articleTitle" json:"articleTitle"`
	ArticleSlug    string     `db:"articleSlug" json:"articleSlug"`
	ArticleExcerpt *string    `db:"articleExcerpt" json:"articleExcerpt"`
	ArticleContent string     `db:"articleContent" json:"articleContent"`
	ArticleImage   *string    `db:"articleImage" json:"articleImage"`
	CategoryID     int64      `db:"categoryID" json:"categoryID"`
	CategoryName   string     `db:"categoryName" json:"categoryName"`
	IsPublished    bool       `db:"isPublished" json:"isPublished"`
	PublishedDate  *time.Time `db:"publishedDate" json:"publishedDate"`
	AuthorID       int64      `db:"authorID" json:"authorID"`
	AuthorName     string     `db:"authorName" json:"authorName"`
	CreatedDate    time.Time  `db:"createdDate" json:"createdDate"`
}

// Category merepresentasikan satu baris lk_article_category.
type Category struct {
	CategoryID   int64  `db:"categoryID" json:"categoryID"`
	CategoryName string `db:"categoryName" json:"categoryName"`
	CategorySlug string `db:"categorySlug" json:"categorySlug"`
	IsActive     bool   `db:"isActive" json:"isActive"`
}
