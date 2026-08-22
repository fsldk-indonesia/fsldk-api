// Package donation_model memuat entitas modul donation. Murni struct data.
package donation_model

import (
	"database/sql"
	"time"
)

// Donation merepresentasikan satu baris tr_donation (dengan kolom join
// judul/slug campaign).
type Donation struct {
	DonationID            int64          `gorm:"column:donationID;primaryKey"`
	PublicRef             string         `gorm:"column:publicRef"`
	CampaignID            int64          `gorm:"column:campaignID"`
	CampaignTitle         string         `gorm:"column:campaignTitle;->"`
	CampaignSlug          string         `gorm:"column:campaignSlug;->"`
	DonorUserID           sql.NullInt64  `gorm:"column:donorUserID"`
	DonorName             string         `gorm:"column:donorName"`
	DonorEmail            string         `gorm:"column:donorEmail"`
	DonorPhone            string         `gorm:"column:donorPhone"`
	DonorAge              sql.NullString `gorm:"column:donorAge"`
	DonorDomicile         sql.NullString `gorm:"column:donorDomicile"`
	DonorOccupation       sql.NullString `gorm:"column:donorOccupation"`
	IsAnonymous           bool           `gorm:"column:isAnonymous"`
	Message               sql.NullString `gorm:"column:message"`
	Amount                float64        `gorm:"column:amount"`
	AdminFee              float64        `gorm:"column:adminFee"`
	TotalAmount           float64        `gorm:"column:totalAmount"`
	PaymentStatus         string         `gorm:"column:paymentStatus"`
	Gateway               string         `gorm:"column:gateway"`
	ExternalTransactionID sql.NullString `gorm:"column:externalTransactionID"`
	IdempotencyKey        string         `gorm:"column:idempotencyKey"`
	QrPayload             sql.NullString `gorm:"column:qrPayload"`
	PaymentCode           sql.NullString `gorm:"column:paymentCode"`
	PaymentLink           sql.NullString `gorm:"column:paymentLink"`
	GatewayStatusID       sql.NullInt64  `gorm:"column:gatewayStatusID"`
	ExpiredDate           sql.NullTime   `gorm:"column:expiredDate"`
	CreatedDate           time.Time      `gorm:"column:createdDate"`
	UpdatedDate           sql.NullTime   `gorm:"column:updatedDate"`
}

// CreateParams menampung data untuk membuat donasi baru (paymentStatus
// selalu PENDING, diset oleh repository — tidak diterima dari caller).
type CreateParams struct {
	PublicRef       string
	CampaignID      int64
	DonorUserID     sql.NullInt64
	DonorName       string
	DonorEmail      string
	DonorPhone      string
	DonorAge        sql.NullString
	DonorDomicile   sql.NullString
	DonorOccupation sql.NullString
	IsAnonymous     bool
	Message         sql.NullString
	Amount          float64
	AdminFee        float64
	TotalAmount     float64
	Gateway         string
	IdempotencyKey  string
	ExpiredDate     sql.NullTime
}
