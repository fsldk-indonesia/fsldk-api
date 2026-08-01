// Package news_dto memuat DTO request/response modul news. Murni struct data.
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

// PublishRequest adalah body mengubah status publikasi berita.
type PublishRequest struct {
	IsPublished bool `json:"isPublished"`
}

// FeaturedRequest adalah body mengubah status unggulan berita.
type FeaturedRequest struct {
	IsFeatured bool `json:"isFeatured"`
}

// Filter menampung parameter penyaringan daftar berita (dipakai repository & service).
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
