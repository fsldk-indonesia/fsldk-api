// Package campaign_model memuat entitas modul campaign. Murni struct data.
package campaign_model

import (
	"database/sql"
	"time"
)

// Campaign merepresentasikan satu baris ms_campaign (dengan kolom join
// kategori/organisasi). beneficiaryName/beneficiaryBankCode/
// beneficiaryAccountNumber/beneficiaryAccountHolder/beneficiaryLockedUntil
// tetap ada di DB tapi tidak lagi dipakai (revisi 2026-09-01 — rekening
// penerima pindah ke withdrawal_request, bukan campaign) — precedent sama
// tr_queue_job, kolom lama dibiarkan menganggur daripada migrasi DROP COLUMN.
type Campaign struct {
	CampaignID               int64          `gorm:"column:campaignID;primaryKey"`
	PublicRef                string         `gorm:"column:publicRef"`
	Slug                     string         `gorm:"column:slug"`
	Title                    string         `gorm:"column:title"`
	CategoryID               int64          `gorm:"column:categoryID"`
	CategoryName             string         `gorm:"column:categoryName;->"`
	OrganizationID           sql.NullInt64  `gorm:"column:organizationID"`
	OrganizationName         sql.NullString `gorm:"column:organizationName;->"`
	ProvinceName             sql.NullString `gorm:"column:provinceName"`
	CityName                 sql.NullString `gorm:"column:cityName"`
	Story                    string         `gorm:"column:story"`
	Goals                    string         `gorm:"column:goals"`
	LatestUpdate             sql.NullString `gorm:"column:latestUpdate"`
	CoverImageUrl            string         `gorm:"column:coverImageUrl"`
	TargetAmount             float64        `gorm:"column:targetAmount"`
	CollectedAmountCache     float64        `gorm:"column:collectedAmountCache"`
	PicName                  string         `gorm:"column:picName"`
	PicPhone                 string         `gorm:"column:picPhone"`
	OrganizationNameOverride sql.NullString `gorm:"column:organizationNameOverride"`
	OrganizationLogoUrl      sql.NullString `gorm:"column:organizationLogoUrl"`
	OrganizationLinkUrl      sql.NullString `gorm:"column:organizationLinkUrl"`
	StartDate                sql.NullTime   `gorm:"column:startDate"`
	EndDate                  sql.NullTime   `gorm:"column:endDate"`
	Status                   string         `gorm:"column:status"`
	ModerationNote           sql.NullString `gorm:"column:moderationNote"`
	IsFeatured               bool           `gorm:"column:isFeatured"`
	IsAnonymousAllowed       bool           `gorm:"column:isAnonymousAllowed"`
	HasDonations             bool           `gorm:"column:hasDonations;->"`
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
	OrganizationID           sql.NullInt64
	ProvinceName             sql.NullString
	CityName                 sql.NullString
	Story                    string
	Goals                    string
	CoverImageUrl            string
	TargetAmount             float64
	PicName                  string
	PicPhone                 string
	OrganizationNameOverride sql.NullString
	OrganizationLogoUrl      sql.NullString
	OrganizationLinkUrl      sql.NullString
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
	ProvinceName             sql.NullString
	CityName                 sql.NullString
	Story                    string
	Goals                    string
	LatestUpdate             sql.NullString
	CoverImageUrl            string
	TargetAmount             float64
	PicName                  string
	PicPhone                 string
	OrganizationNameOverride sql.NullString
	OrganizationLogoUrl      sql.NullString
	OrganizationLinkUrl      sql.NullString
	StartDate                sql.NullTime
	EndDate                  sql.NullTime
	IsAnonymousAllowed       bool
	UpdatedBy                int64
}
