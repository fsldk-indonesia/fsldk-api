// Package withdrawal_model memuat entitas modul withdrawal. Murni struct data.
package withdrawal_model

import (
	"database/sql"
	"time"
)

// Withdrawal merepresentasikan satu baris tr_withdrawal (dengan kolom join
// judul campaign).
type Withdrawal struct {
	WithdrawalID             int64          `gorm:"column:withdrawalID;primaryKey"`
	WithdrawalRef            string         `gorm:"column:withdrawalRef"`
	CampaignID               int64          `gorm:"column:campaignID"`
	CampaignTitle            string         `gorm:"column:campaignTitle;->"`
	RequestedByUserID        int64          `gorm:"column:requestedByUserID"`
	Amount                   float64        `gorm:"column:amount"`
	Fee                      float64        `gorm:"column:fee"`
	NetAmount                float64        `gorm:"column:netAmount"`
	BeneficiaryBankCode      string         `gorm:"column:beneficiaryBankCode"`
	BeneficiaryAccountNumber string         `gorm:"column:beneficiaryAccountNumber"`
	BeneficiaryAccountHolder string         `gorm:"column:beneficiaryAccountHolder"`
	Status                   string         `gorm:"column:status"`
	ApprovedByUserID         sql.NullInt64  `gorm:"column:approvedByUserID"`
	ApprovedDate             sql.NullTime   `gorm:"column:approvedDate"`
	RejectionReason          sql.NullString `gorm:"column:rejectionReason"`
	IdempotencyKey           string         `gorm:"column:idempotencyKey"`
	GatewayStatusID          sql.NullInt64  `gorm:"column:gatewayStatusID"`
	GatewayResponseJSON      sql.NullString `gorm:"column:gatewayResponseJSON"`
	ExecutedDate             sql.NullTime   `gorm:"column:executedDate"`
	CompletedDate            sql.NullTime   `gorm:"column:completedDate"`
	CreatedDate              time.Time      `gorm:"column:createdDate"`
	UpdatedDate              sql.NullTime   `gorm:"column:updatedDate"`
}

// CreateParams menampung data untuk membuat withdrawal baru (status selalu
// REQUESTED lalu langsung SECURITY_CHECK — diset oleh service, bukan diterima
// dari caller).
type CreateParams struct {
	WithdrawalRef            string
	CampaignID               int64
	RequestedByUserID        int64
	Amount                   float64
	Fee                      float64
	NetAmount                float64
	BeneficiaryBankCode      string
	BeneficiaryAccountNumber string
	BeneficiaryAccountHolder string
	IdempotencyKey           string
}

// StatusUpdateParams menampung field opsional yang berubah pada satu
// transisi status withdrawal — satu method repository generik dipakai
// seluruh transisi (approve/reject/process/callback/cancel) alih-alih
// method sempit per transisi.
type StatusUpdateParams struct {
	ApprovedByUserID    *int64
	RejectionReason     *string
	GatewayStatusID     *int
	GatewayResponseJSON *string
	SetExecutedNow      bool
	SetCompletedNow     bool
}
