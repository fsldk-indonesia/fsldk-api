// Package article_dto memuat DTO request/response modul article. Murni struct data.
package article_dto

// Request adalah body membuat/memperbarui artikel.
type Request struct {
	ArticleTitle   string `json:"articleTitle" validate:"required,min=3,max=255"`
	ArticleExcerpt string `json:"articleExcerpt" validate:"max=500"`
	ArticleContent string `json:"articleContent" validate:"required"`
	ArticleImage   string `json:"articleImage" validate:"max=255"`
	CategoryID     int64  `json:"categoryID" validate:"required"`
	Status         string `json:"status" validate:"omitempty,oneof=draft published"`
}

// PublishRequest adalah body mengubah status publikasi artikel.
type PublishRequest struct {
	IsPublished bool `json:"isPublished"`
}

// Filter menampung parameter penyaringan daftar artikel (dipakai repository & service).
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
