// Package article_dto memuat DTO request/response modul article. Murni struct data.
package article_dto

// Request adalah body membuat/memperbarui artikel.
type Request struct {
	ArticleTitle  string `json:"articleTitle" validate:"required,min=3,max=255"`
	ArticleIntro  string `json:"articleIntro" validate:"required"`
	ArticleImage  string `json:"articleImage" validate:"max=255"`
	ArticleWriter string `json:"articleWriter" validate:"required,min=2,max=150"`
	ArticleEditor string `json:"articleEditor" validate:"max=150"`
	ArticlePdf    string `json:"articlePdf" validate:"max=255"`
	CategoryID    int64  `json:"categoryID" validate:"required"`
	Status        string `json:"status" validate:"omitempty,oneof=draft published"`
}

// PublishRequest adalah body mengubah status publikasi artikel.
type PublishRequest struct {
	IsPublished bool `json:"isPublished"`
}

// Filter menampung parameter penyaringan daftar artikel (dipakai repository & service).
type Filter struct {
	Search        string
	Writer        string // LIKE terhadap articleWriter — filter kolom "Penulis" CMS
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
	Writer       string
	CategoryName string
	DateFrom     string
	DateTo       string
}

// BulkDeleteRequest adalah body untuk menghapus banyak artikel sekaligus.
type BulkDeleteRequest struct {
	IDs []int64 `json:"ids" validate:"required,min=1"`
}
