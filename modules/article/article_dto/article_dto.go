// Package article_dto memuat DTO request/response modul article.
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
