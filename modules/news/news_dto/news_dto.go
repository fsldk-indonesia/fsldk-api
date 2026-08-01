// Package news_dto memuat DTO request/response modul news.
package news_dto

// Request adalah body membuat/memperbarui berita.
type Request struct {
	NewsTitle   string `json:"newsTitle" validate:"required,min=3,max=255"`
	NewsExcerpt string `json:"newsExcerpt" validate:"max=500"`
	NewsContent string `json:"newsContent" validate:"required"`
	NewsImage   string `json:"newsImage" validate:"max=255"`
	CategoryID  int64  `json:"categoryID" validate:"required"`
	IsFeatured  bool   `json:"isFeatured"`
	Status      string `json:"status" validate:"omitempty,oneof=draft published"`
}
