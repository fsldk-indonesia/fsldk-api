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
	Writer        string   // LIKE terhadap articleWriter — filter kolom "Penulis" CMS
	Writers       []string // exact match (IN) terhadap articleWriter — filter "Penulis" publik, multi-select dari nilai distinct
	CategorySlugs []string // exact match (IN) — filter kategori publik, multi-select by slug
	CategoryIDs   []int64  // exact match (IN) — filter kolom "Kategori" CMS, multi-select by ID
	Years         []int    // exact match (IN) terhadap YEAR(publishedDate) — filter "Tahun Publikasi" publik
	Months        []int    // exact match (IN) terhadap MONTH(publishedDate) — filter "Bulan Publikasi" publik
	HasPdf        *bool    // nil = semua, true = articlePdf terisi, false = articlePdf kosong — filter "Punya PDF" publik
	PublishedOnly bool
	Status        []string // "published" | "draft" — multi-select (IN), kosong = semua status
	DateFrom      string   // "YYYY-MM-DD", inklusif — filter kolom "Tanggal" (createdDate) CMS
	DateTo        string   // "YYYY-MM-DD", inklusif
	Limit         int
	Offset        int
	OrderBy       string
}

// PublicFilter menampung parameter filter khusus endpoint publik List — dipisah
// dari Filter (dipakai repository) dengan alasan yang sama dengan CMSFilter:
// supaya signature service.PublicList tidak terus bertambah parameter
// positional setiap kali kolom filter baru ditambahkan.
type PublicFilter struct {
	CategorySlugs []string
	Years         []int
	Writers       []string
	Months        []int
	HasPdf        *bool
}

// FilterOptionsResponse menampung nilai distinct untuk mengisi dropdown filter
// publik "Tahun Publikasi" & "Penulis" — kategori sudah punya endpoint sendiri
// (Categories), tidak diulang di sini. Nilainya distinct dari data yang benar-benar
// ada, supaya dropdown tidak pernah menawarkan pilihan yang hasilnya kosong.
type FilterOptionsResponse struct {
	Years   []int    `json:"years"`
	Writers []string `json:"writers"`
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
