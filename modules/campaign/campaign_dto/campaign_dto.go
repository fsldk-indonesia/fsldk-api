// Package campaign_dto memuat DTO request/response modul campaign. Murni struct data.
package campaign_dto

import "time"

// CreateRequest adalah body membuat campaign baru (selalu berstatus DRAFT).
// Rekening penerima TIDAK lagi bagian dari campaign (revisi 2026-09-01) —
// pindah ke saat pengajuan withdrawal (diinput ulang setiap kali, tidak
// disimpan di campaign). Field mengikuti persis Campaign celengan syahid
// (ldksyahid-app): provinsi/kota bebas teks, tujuan terpisah dari cerita,
// PIC (nama_pj/telp_pj) menggantikan owner sebagai target notifikasi WA.
type CreateRequest struct {
	Title                    string   `json:"title" validate:"required,min=5,max=200"`
	CategoryID               int64    `json:"categoryID" validate:"required"`
	OrganizationID           *int64   `json:"organizationID"`
	ProvinceName             string   `json:"provinceName" validate:"max=100"`
	CityName                 string   `json:"cityName" validate:"max=100"`
	Story                    string   `json:"story" validate:"required,min=50"`
	Goals                    string   `json:"goals" validate:"required,min=10"`
	CoverImageUrl            string   `json:"coverImageUrl" validate:"required,max=500"`
	SupportingImageUrls      []string `json:"supportingImageUrls" validate:"max=10,dive,max=500"`
	TargetAmount             float64  `json:"targetAmount" validate:"required,gt=0"`
	PicName                  string   `json:"picName" validate:"required,max=150"`
	PicPhone                 string   `json:"picPhone" validate:"required,max=30"`
	OrganizationNameOverride string   `json:"organizationNameOverride" validate:"max=150"`
	OrganizationLogoUrl      string   `json:"organizationLogoUrl" validate:"max=500"`
	OrganizationLinkUrl      string   `json:"organizationLinkUrl" validate:"max=500"`
	StartDate                *string  `json:"startDate"` // format YYYY-MM-DDTHH:mm (atau YYYY-MM-DD, tanpa jam)
	EndDate                  *string  `json:"endDate"`   // format YYYY-MM-DDTHH:mm (atau YYYY-MM-DD, tanpa jam)
	IsAnonymousAllowed       *bool    `json:"isAnonymousAllowed"`
}

// UpdateRequest adalah body memperbarui campaign — campaign murni CRUD di
// CMS, berlaku pada status apapun kecuali ARCHIVED. Field sama dengan
// CreateRequest; SupportingImageUrls bernilai nil berarti "tidak diubah"
// (dibedakan dari slice kosong yang berarti "hapus seluruh gambar
// pendukung"), memanfaatkan semantik nil-vs-empty JSON standar Go.
type UpdateRequest struct {
	Title                    string   `json:"title" validate:"required,min=5,max=200"`
	CategoryID               int64    `json:"categoryID" validate:"required"`
	OrganizationID           *int64   `json:"organizationID"`
	ProvinceName             string   `json:"provinceName" validate:"max=100"`
	CityName                 string   `json:"cityName" validate:"max=100"`
	Story                    string   `json:"story" validate:"required,min=50"`
	Goals                    string   `json:"goals" validate:"required,min=10"`
	CoverImageUrl            string   `json:"coverImageUrl" validate:"required,max=500"`
	SupportingImageUrls      []string `json:"supportingImageUrls" validate:"max=10,dive,max=500"`
	TargetAmount             float64  `json:"targetAmount" validate:"required,gt=0"`
	LatestUpdate             string   `json:"latestUpdate" validate:"max=5000"`
	PicName                  string   `json:"picName" validate:"required,max=150"`
	PicPhone                 string   `json:"picPhone" validate:"required,max=30"`
	OrganizationNameOverride string   `json:"organizationNameOverride" validate:"max=150"`
	OrganizationLogoUrl      string   `json:"organizationLogoUrl" validate:"max=500"`
	OrganizationLinkUrl      string   `json:"organizationLinkUrl" validate:"max=500"`
	StartDate                *string  `json:"startDate"`
	EndDate                  *string  `json:"endDate"`
	IsAnonymousAllowed       *bool    `json:"isAnonymousAllowed"`
}

// ListFilter menampung parameter penyaringan daftar campaign.
type ListFilter struct {
	Status     string
	CategoryID int64
	Search     string
	Limit      int
	Offset     int
	OrderBy    string
}

// Response adalah representasi ringkas campaign untuk listing & mutasi.
type Response struct {
	CampaignID               int64      `json:"campaignID"`
	PublicRef                string     `json:"publicRef"`
	Slug                     string     `json:"slug"`
	Title                    string     `json:"title"`
	CategoryID               int64      `json:"categoryID"`
	CategoryName             string     `json:"categoryName"`
	OrganizationID           *int64     `json:"organizationID"`
	OrganizationName         *string    `json:"organizationName"`
	ProvinceName             string     `json:"provinceName,omitempty"`
	CityName                 string     `json:"cityName,omitempty"`
	Story                    string     `json:"story"`
	Goals                    string     `json:"goals"`
	LatestUpdate             string     `json:"latestUpdate,omitempty"`
	CoverImageUrl            string     `json:"coverImageUrl"`
	TargetAmount             float64    `json:"targetAmount"`
	CollectedAmount          float64    `json:"collectedAmount"`
	PicName                  string     `json:"picName"`
	PicPhone                 string     `json:"picPhone"`
	OrganizationNameOverride string     `json:"organizationNameOverride,omitempty"`
	OrganizationLogoUrl      string     `json:"organizationLogoUrl,omitempty"`
	OrganizationLinkUrl      string     `json:"organizationLinkUrl,omitempty"`
	StartDate                *time.Time `json:"startDate"`
	EndDate                  *time.Time `json:"endDate"`
	Status                   string     `json:"status"`
	ModerationNote           string     `json:"moderationNote,omitempty"`
	IsFeatured               bool       `json:"isFeatured"`
	IsAnonymousAllowed       bool       `json:"isAnonymousAllowed"`
	HasDonations             bool       `json:"hasDonations"`
	CreatedDate              time.Time  `json:"createdDate"`
}

// DetailResponse adalah Response ditambah gambar pendukung — dipakai
// endpoint detail/mutasi single-record.
type DetailResponse struct {
	Response
	SupportingImageUrls []string `json:"supportingImageUrls"`
}

// CategoryResponse adalah satu baris lk_campaign_category.
type CategoryResponse struct {
	CampaignCategoryID int64  `json:"campaignCategoryID"`
	CategoryCode       string `json:"categoryCode"`
	CategoryName       string `json:"categoryName"`
}

// LiteResponse adalah representasi campaign paling ringkas (id + judul) —
// dipakai populasi dropdown campaign di Laporan Kantong Amal (item 6).
type LiteResponse struct {
	CampaignID int64  `json:"campaignID"`
	Title      string `json:"title"`
}
