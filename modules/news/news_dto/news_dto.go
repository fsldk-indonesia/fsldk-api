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
	Reporter      string  // LIKE terhadap newsReporter — filter kolom "Reporter" CMS
	CategorySlug  string
	CategoryIDs   []int64 // exact match (IN) — filter kolom "Kategori" CMS, multi-select by ID
	PublishedOnly bool
	Status        []string // "published" | "draft" — multi-select (IN), kosong = semua status
	DateFrom      string   // "YYYY-MM-DD", inklusif — filter kolom "Tanggal" (createdDate) CMS
	DateTo        string   // "YYYY-MM-DD", inklusif
	// Years/Reporters/IsFeatured dipakai dropdown filter PUBLIK "Filter Berita"
	// (beda dari Reporter/DateFrom/DateTo di atas yang khusus CMS) — exact
	// match (IN), mendukung pilih lebih dari satu nilai sekaligus.
	Years      []int64  // filter YEAR(COALESCE(publishedDate, createdDate)) IN (...)
	Reporters  []string // exact match (IN) terhadap newsReporter, BUKAN LIKE seperti Reporter CMS
	IsFeatured *bool    // nil = tidak difilter, else true/false eksak
	Limit      int
	Offset     int
	OrderBy    string
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
	Reporter    string
	DateFrom    string
	DateTo      string
}

// PublicFilter menampung parameter filter dropdown publik "Filter Berita" di
// luar kategori (yang sudah jadi parameter terpisah sejak awal pada
// Service.PublicList) — Tahun Terbit, Penulis, dan status Unggulan.
type PublicFilter struct {
	Years      []int64
	Reporters  []string
	IsFeatured *bool
}

// FilterOptionsResponse menampung nilai distinct yang mengisi dropdown filter
// publik (Tahun Terbit/Penulis) — supaya dropdown tidak pernah menawarkan
// pilihan yang hasilnya kosong, pola sama seperti gallery_dto.FilterOptionsResponse.
type FilterOptionsResponse struct {
	Years     []int64  `json:"years"`
	Reporters []string `json:"reporters"`
}

// BulkDeleteRequest adalah body untuk menghapus banyak berita sekaligus.
type BulkDeleteRequest struct {
	IDs []int64 `json:"ids" validate:"required,min=1"`
}
