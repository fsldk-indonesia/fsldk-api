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
	PaymentMethod         sql.NullString `gorm:"column:paymentMethod"`
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

// AdminCreateParams menampung data membuat donasi manual/offline
// (gateway="manual", tidak pernah menyentuh tr_wallet_ledger — lihat
// constants.DonationGatewayManual) — dipakai admin mencatat donasi yang
// tidak lewat Amdigipay/Bisatopup (item 1 revision-prompt-2.md).
type AdminCreateParams struct {
	PublicRef       string
	CampaignID      int64
	DonorName       string
	DonorEmail      sql.NullString
	DonorPhone      sql.NullString
	DonorAge        sql.NullString
	DonorDomicile   sql.NullString
	DonorOccupation sql.NullString
	IsAnonymous     bool
	Message         sql.NullString
	Amount          float64
	PaymentMethod   sql.NullString
	PaymentStatus   string
	IdempotencyKey  string
}

// AdminUpdateParams menampung data mengedit donasi manual/offline — hanya
// berlaku untuk baris gateway="manual" (divalidasi di donation_service).
type AdminUpdateParams struct {
	DonorName       string
	DonorEmail      sql.NullString
	DonorPhone      sql.NullString
	DonorAge        sql.NullString
	DonorDomicile   sql.NullString
	DonorOccupation sql.NullString
	IsAnonymous     bool
	Message         sql.NullString
	Amount          float64
	PaymentMethod   sql.NullString
	PaymentStatus   string
}

// GatewayResultParams menampung hasil CreateQRISTransaction untuk disimpan
// ke donasi yang sudah dibuat (dipanggil segera setelah repo.Create sukses).
type GatewayResultParams struct {
	ExternalTransactionID string
	QrPayload             string
	PaymentCode           string
	PaymentLink           string
}

// CallbackUpdateParams menampung perubahan status donasi dari callback
// payment gateway. TotalAmount/AdminFee nil berarti tidak diubah — dipakai
// saat status akhirnya bukan PAID atau saat AMOUNT_MISMATCH (nilai asli
// dipertahankan untuk resolusi manual, tidak dipercaya begitu saja dari gateway).
type CallbackUpdateParams struct {
	PaymentStatus   string
	GatewayStatusID int
	TotalAmount     *float64
	AdminFee        *float64
}
