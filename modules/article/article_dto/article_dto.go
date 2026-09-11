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
	Writer        string  // LIKE terhadap articleWriter — filter kolom "Penulis" CMS
	CategorySlug  string
	CategoryIDs   []int64 // exact match (IN) — filter kolom "Kategori" CMS, multi-select by ID
	PublishedOnly bool
	Status        []string // "published" | "draft" — multi-select (IN), kosong = semua status
	DateFrom      string   // "YYYY-MM-DD", inklusif — filter kolom "Tanggal" (createdDate) CMS
	DateTo        string   // "YYYY-MM-DD", inklusif
	Limit         int
	Offset        int
	OrderBy       string
}

// CMSFilter menampung parameter filter khusus endpoint CMS list (di luar
// dto.ListQuery yang sudah menampung search/page/limit/sort) — dipisah dari
// Filter (dipakai repository) supaya signature service.CMSList tidak terus
// bertambah parameter positional setiap kali kolom filter baru ditambahkan.
// Status/CategoryIDs multi-select (checkbox) — dikirim frontend sebagai query
// param comma-separated, di-parse handler lewat dto.ParseCSV/ParseInt64CSV.
type CMSFilter struct {
	Status      []string
	CategoryIDs []int64
	Writer      string
	DateFrom    string
	DateTo      string
}

// BulkDeleteRequest adalah body untuk menghapus banyak artikel sekaligus.
type BulkDeleteRequest struct {
	IDs []int64 `json:"ids" validate:"required,min=1"`
}
