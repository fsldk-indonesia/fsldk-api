// Package campaign_model memuat entitas modul campaign. Murni struct data.
package campaign_model

import (
	"database/sql"
	"time"
)

// Campaign merepresentasikan satu baris ms_campaign (dengan kolom join
// kategori/pemilik/organisasi).
type Campaign struct {
	CampaignID               int64          `gorm:"column:campaignID;primaryKey"`
	PublicRef                string         `gorm:"column:publicRef"`
	Slug                     string         `gorm:"column:slug"`
	Title                    string         `gorm:"column:title"`
	CategoryID               int64          `gorm:"column:categoryID"`
	CategoryName             string         `gorm:"column:categoryName;->"`
	OwnerUserID              int64          `gorm:"column:ownerUserID"`
	OwnerName                string         `gorm:"column:ownerName;->"`
	OrganizationID           sql.NullInt64  `gorm:"column:organizationID"`
	OrganizationName         sql.NullString `gorm:"column:organizationName;->"`
	Story                    string         `gorm:"column:story"`
	LatestUpdate             sql.NullString `gorm:"column:latestUpdate"`
	CoverImageUrl            string         `gorm:"column:coverImageUrl"`
	TargetAmount             float64        `gorm:"column:targetAmount"`
	CollectedAmountCache     float64        `gorm:"column:collectedAmountCache"`
	BeneficiaryName          string         `gorm:"column:beneficiaryName"`
	BeneficiaryBankCode      string         `gorm:"column:beneficiaryBankCode"`
	BeneficiaryAccountNumber string         `gorm:"column:beneficiaryAccountNumber"`
	BeneficiaryAccountHolder string         `gorm:"column:beneficiaryAccountHolder"`
	BeneficiaryLockedUntil   sql.NullTime   `gorm:"column:beneficiaryLockedUntil"`
	StartDate                sql.NullTime   `gorm:"column:startDate"`
	EndDate                  sql.NullTime   `gorm:"column:endDate"`
	Status                   string         `gorm:"column:status"`
	ModerationNote           sql.NullString `gorm:"column:moderationNote"`
	IsFeatured               bool           `gorm:"column:isFeatured"`
	IsAnonymousAllowed       bool           `gorm:"column:isAnonymousAllowed"`
	CreatedDate              time.Time      `gorm:"column:createdDate"`
	CreatedBy                int64          `gorm:"column:createdBy"`
	UpdatedDate              sql.NullTime   `gorm:"column:updatedDate"`
	UpdatedBy                sql.NullInt64  `gorm:"column:updatedBy"`
}

// Image merepresentasikan satu baris ms_campaign_image.
type Image struct {
	CampaignImageID int64  `gorm:"column:campaignImageID;primaryKey"`
	CampaignID      int64  `gorm:"column:campaignID"`
	ImageUrl        string `gorm:"column:imageUrl"`
	SortOrder       int    `gorm:"column:sortOrder"`
}

// Review merepresentasikan satu baris tr_campaign_review (dengan nama reviewer, hasil join).
type Review struct {
	ReviewID       int64          `gorm:"column:reviewID;primaryKey"`
	CampaignID     int64          `gorm:"column:campaignID"`
	ReviewerUserID int64          `gorm:"column:reviewerUserID"`
	ReviewerName   string         `gorm:"column:reviewerName;->"`
	Decision       string         `gorm:"column:decision"`
	Note           sql.NullString `gorm:"column:note"`
	ReviewedDate   time.Time      `gorm:"column:reviewedDate"`
}

// Category merepresentasikan satu baris lk_campaign_category.
type Category struct {
	CampaignCategoryID int64  `gorm:"column:campaignCategoryID;primaryKey"`
	CategoryCode       string `gorm:"column:categoryCode"`
	CategoryName       string `gorm:"column:categoryName"`
	SortOrder          int    `gorm:"column:sortOrder"`
}

// CreateParams menampung data untuk membuat campaign baru (status selalu
// DRAFT, diset oleh repository — tidak diterima dari caller).
type CreateParams struct {
	PublicRef                string
	Slug                     string
	Title                    string
	CategoryID               int64
	OwnerUserID              int64
	OrganizationID           sql.NullInt64
	Story                    string
	CoverImageUrl            string
	TargetAmount             float64
	BeneficiaryName          string
	BeneficiaryBankCode      string
	BeneficiaryAccountNumber string
	BeneficiaryAccountHolder string
	StartDate                sql.NullTime
	EndDate                  sql.NullTime
	IsAnonymousAllowed       bool
	CreatedBy                int64
}

// UpdateParams menampung data untuk memperbarui campaign existing.
type UpdateParams struct {
	Slug                     string
	Title                    string
	CategoryID               int64
	OrganizationID           sql.NullInt64
	Story                    string
	LatestUpdate             sql.NullString
	CoverImageUrl            string
	TargetAmount             float64
	BeneficiaryName          string
	BeneficiaryBankCode      string
	BeneficiaryAccountNumber string
	BeneficiaryAccountHolder string
	StartDate                sql.NullTime
	EndDate                  sql.NullTime
	IsAnonymousAllowed       bool
	UpdatedBy                int64
}

// ReviewParams menampung data untuk mencatat satu baris tr_campaign_review.
type ReviewParams struct {
	CampaignID     int64
	ReviewerUserID int64
	Decision       string
	Note           sql.NullString
}
