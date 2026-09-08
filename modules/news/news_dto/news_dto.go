// Package news_dto memuat DTO request/response modul news. Murni struct data.
package news_dto

// Request adalah body membuat/memperbarui berita.
type Request struct {
	NewsTitle     string `json:"newsTitle" validate:"required,min=3,max=255"`
	NewsExcerpt   string `json:"newsExcerpt" validate:"max=500"`
	NewsContent   string `json:"newsContent" validate:"required"`
	NewsImage     string `json:"newsImage" validate:"max=255"`
	NewsPublisher string `json:"newsPublisher" validate:"max=150"`
	NewsReporter  string `json:"newsReporter" validate:"required,min=2,max=150"`
	NewsEditor    string `json:"newsEditor" validate:"max=150"`
	CategoryID    int64  `json:"categoryID" validate:"required"`
	IsFeatured    bool   `json:"isFeatured"`
	Status        string `json:"status" validate:"omitempty,oneof=draft published"`
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
	Reporter      string // LIKE terhadap newsReporter — filter kolom "Reporter" CMS
	CategoryName  string // LIKE terhadap categoryName — filter kolom "Kategori" CMS (free text, bukan dropdown categoryID)
	CategorySlug  string
	CategoryID    int64
	PublishedOnly bool
	Status        string // "published" | "draft" | ""
	DateFrom      string // "YYYY-MM-DD", inklusif — filter kolom "Tanggal" (createdDate) CMS
	DateTo        string // "YYYY-MM-DD", inklusif
	Limit         int
	Offset        int
	OrderBy       string
}

// CMSFilter menampung parameter filter khusus endpoint CMS list (di luar
// dto.ListQuery yang sudah menampung search/page/limit/sort) — dipisah dari
// Filter (dipakai repository) supaya signature service.CMSList tidak terus
// bertambah parameter positional setiap kali kolom filter baru ditambahkan.
type CMSFilter struct {
	Status       string
	CategoryID   int64
	Reporter     string
	CategoryName string
	DateFrom     string
	DateTo       string
}

// BulkDeleteRequest adalah body untuk menghapus banyak berita sekaligus.
type BulkDeleteRequest struct {
	IDs []int64 `json:"ids" validate:"required,min=1"`
}
