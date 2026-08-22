// Package campaign_dto memuat DTO request/response modul campaign. Murni struct data.
package campaign_dto

import "time"

// CreateRequest adalah body membuat campaign baru (selalu berstatus DRAFT).
type CreateRequest struct {
	Title                    string   `json:"title" validate:"required,min=5,max=200"`
	CategoryID               int64    `json:"categoryID" validate:"required"`
	OrganizationID           *int64   `json:"organizationID"`
	Story                    string   `json:"story" validate:"required,min=50"`
	CoverImageUrl            string   `json:"coverImageUrl" validate:"required,max=500"`
	SupportingImageUrls      []string `json:"supportingImageUrls" validate:"max=10,dive,max=500"`
	TargetAmount             float64  `json:"targetAmount" validate:"required,gt=0"`
	BeneficiaryName          string   `json:"beneficiaryName" validate:"required,max=150"`
	BeneficiaryBankCode      string   `json:"beneficiaryBankCode" validate:"required,max=20"`
	BeneficiaryAccountNumber string   `json:"beneficiaryAccountNumber" validate:"required,max=30"`
	BeneficiaryAccountHolder string   `json:"beneficiaryAccountHolder" validate:"required,max=150"`
	StartDate                *string  `json:"startDate"` // format YYYY-MM-DD
	EndDate                  *string  `json:"endDate"`   // format YYYY-MM-DD
	IsAnonymousAllowed       *bool    `json:"isAnonymousAllowed"`
}

// UpdateRequest adalah body memperbarui campaign — hanya berlaku saat
// campaign berstatus DRAFT/REVISION_REQUESTED. Field sama dengan
// CreateRequest; SupportingImageUrls bernilai nil berarti "tidak diubah"
// (dibedakan dari slice kosong yang berarti "hapus seluruh gambar
// pendukung"), memanfaatkan semantik nil-vs-empty JSON standar Go.
type UpdateRequest struct {
	Title                    string   `json:"title" validate:"required,min=5,max=200"`
	CategoryID               int64    `json:"categoryID" validate:"required"`
	OrganizationID           *int64   `json:"organizationID"`
	Story                    string   `json:"story" validate:"required,min=50"`
	CoverImageUrl            string   `json:"coverImageUrl" validate:"required,max=500"`
	SupportingImageUrls      []string `json:"supportingImageUrls" validate:"max=10,dive,max=500"`
	TargetAmount             float64  `json:"targetAmount" validate:"required,gt=0"`
	LatestUpdate             string   `json:"latestUpdate" validate:"max=5000"`
	BeneficiaryName          string   `json:"beneficiaryName" validate:"required,max=150"`
	BeneficiaryBankCode      string   `json:"beneficiaryBankCode" validate:"required,max=20"`
	BeneficiaryAccountNumber string   `json:"beneficiaryAccountNumber" validate:"required,max=30"`
	BeneficiaryAccountHolder string   `json:"beneficiaryAccountHolder" validate:"required,max=150"`
	StartDate                *string  `json:"startDate"`
	EndDate                  *string  `json:"endDate"`
	IsAnonymousAllowed       *bool    `json:"isAnonymousAllowed"`
}

// ReviewRequest adalah body moderasi campaign oleh reviewer CMS.
type ReviewRequest struct {
	Decision string `json:"decision" validate:"required,oneof=APPROVED REVISION_REQUESTED REJECTED"`
	Note     string `json:"note" validate:"max=1000"`
}

// ListFilter menampung parameter penyaringan daftar campaign.
type ListFilter struct {
	Status      string
	CategoryID  int64
	OwnerUserID *int64
	Search      string
	Limit       int
	Offset      int
	OrderBy     string
}

// Response adalah representasi ringkas campaign untuk listing & mutasi.
type Response struct {
	CampaignID               int64      `json:"campaignID"`
	PublicRef                string     `json:"publicRef"`
	Slug                     string     `json:"slug"`
	Title                    string     `json:"title"`
	CategoryID               int64      `json:"categoryID"`
	CategoryName             string     `json:"categoryName"`
	OwnerUserID              int64      `json:"ownerUserID"`
	OwnerName                string     `json:"ownerName"`
	OrganizationID           *int64     `json:"organizationID"`
	OrganizationName         *string    `json:"organizationName"`
	Story                    string     `json:"story"`
	LatestUpdate             string     `json:"latestUpdate,omitempty"`
	CoverImageUrl            string     `json:"coverImageUrl"`
	TargetAmount             float64    `json:"targetAmount"`
	CollectedAmount          float64    `json:"collectedAmount"`
	BeneficiaryName          string     `json:"beneficiaryName"`
	BeneficiaryBankCode      string     `json:"beneficiaryBankCode"`
	BeneficiaryAccountNumber string     `json:"beneficiaryAccountNumber"`
	BeneficiaryAccountHolder string     `json:"beneficiaryAccountHolder"`
	StartDate                *time.Time `json:"startDate"`
	EndDate                  *time.Time `json:"endDate"`
	Status                   string     `json:"status"`
	ModerationNote           string     `json:"moderationNote,omitempty"`
	IsFeatured               bool       `json:"isFeatured"`
	IsAnonymousAllowed       bool       `json:"isAnonymousAllowed"`
	CreatedDate              time.Time  `json:"createdDate"`
}

// DetailResponse adalah Response ditambah gambar pendukung — dipakai
// endpoint detail/mutasi single-record.
type DetailResponse struct {
	Response
	SupportingImageUrls []string `json:"supportingImageUrls"`
}

// ReviewResponse adalah satu baris riwayat moderasi campaign.
type ReviewResponse struct {
	ReviewID     int64     `json:"reviewID"`
	ReviewerName string    `json:"reviewerName"`
	Decision     string    `json:"decision"`
	Note         string    `json:"note,omitempty"`
	ReviewedDate time.Time `json:"reviewedDate"`
}

// CategoryResponse adalah satu baris lk_campaign_category.
type CategoryResponse struct {
	CampaignCategoryID int64  `json:"campaignCategoryID"`
	CategoryCode       string `json:"categoryCode"`
	CategoryName       string `json:"categoryName"`
}
