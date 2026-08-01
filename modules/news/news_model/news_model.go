// Package news_model memuat entitas modul news. Murni struct data.
package news_model

import "time"

// News merepresentasikan satu baris ms_news (dengan kolom join kategori & penulis).
type News struct {
	NewsID        int64      `gorm:"column:newsID;primaryKey" json:"newsID"`
	NewsTitle     string     `gorm:"column:newsTitle" json:"newsTitle"`
	NewsSlug      string     `gorm:"column:newsSlug" json:"newsSlug"`
	NewsExcerpt   *string    `gorm:"column:newsExcerpt" json:"newsExcerpt"`
	NewsContent   string     `gorm:"column:newsContent" json:"newsContent"`
	NewsImage     *string    `gorm:"column:newsImage" json:"newsImage"`
	CategoryID    int64      `gorm:"column:categoryID" json:"categoryID"`
	CategoryName  string     `gorm:"column:categoryName;->" json:"categoryName"`
	IsFeatured    bool       `gorm:"column:isFeatured" json:"isFeatured"`
	IsPublished   bool       `gorm:"column:isPublished" json:"isPublished"`
	PublishedDate *time.Time `gorm:"column:publishedDate" json:"publishedDate"`
	ViewCount     int64      `gorm:"column:viewCount" json:"viewCount"`
	AuthorID      int64      `gorm:"column:authorID" json:"authorID"`
	AuthorName    string     `gorm:"column:authorName;->" json:"authorName"`
	CreatedDate   time.Time  `gorm:"column:createdDate" json:"createdDate"`
}

// Category merepresentasikan satu baris lk_news_category.
type Category struct {
	CategoryID   int64  `gorm:"column:categoryID;primaryKey" json:"categoryID"`
	CategoryName string `gorm:"column:categoryName" json:"categoryName"`
	CategorySlug string `gorm:"column:categorySlug" json:"categorySlug"`
	IsActive     bool   `gorm:"column:isActive" json:"isActive"`
}
