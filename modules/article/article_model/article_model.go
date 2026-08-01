// Package article_model memuat entitas modul article. Murni struct data.
package article_model

import "time"

// Article merepresentasikan satu baris ms_article.
type Article struct {
	ArticleID      int64      `gorm:"column:articleID;primaryKey" json:"articleID"`
	ArticleTitle   string     `gorm:"column:articleTitle" json:"articleTitle"`
	ArticleSlug    string     `gorm:"column:articleSlug" json:"articleSlug"`
	ArticleExcerpt *string    `gorm:"column:articleExcerpt" json:"articleExcerpt"`
	ArticleContent string     `gorm:"column:articleContent" json:"articleContent"`
	ArticleImage   *string    `gorm:"column:articleImage" json:"articleImage"`
	CategoryID     int64      `gorm:"column:categoryID" json:"categoryID"`
	CategoryName   string     `gorm:"column:categoryName;->" json:"categoryName"`
	IsPublished    bool       `gorm:"column:isPublished" json:"isPublished"`
	PublishedDate  *time.Time `gorm:"column:publishedDate" json:"publishedDate"`
	AuthorID       int64      `gorm:"column:authorID" json:"authorID"`
	AuthorName     string     `gorm:"column:authorName;->" json:"authorName"`
	CreatedDate    time.Time  `gorm:"column:createdDate" json:"createdDate"`
}

// Category merepresentasikan satu baris lk_article_category.
type Category struct {
	CategoryID   int64  `gorm:"column:categoryID;primaryKey" json:"categoryID"`
	CategoryName string `gorm:"column:categoryName" json:"categoryName"`
	CategorySlug string `gorm:"column:categorySlug" json:"categorySlug"`
	IsActive     bool   `gorm:"column:isActive" json:"isActive"`
}
