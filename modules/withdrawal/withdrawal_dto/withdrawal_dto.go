// Package withdrawal_dto memuat DTO request/response modul withdrawal.
// Murni struct data.
package withdrawal_dto

import "time"

// CreateRequest adalah body mengajukan penarikan saldo baru. Beneficiary
// selalu memakai rekening default campaign (ms_campaign) — mengganti
// rekening tujuan bukan bagian dari request ini (lihat 11-withdrawal.md
// §11.3, mekanisme ganti rekening + cooling period dibangun terpisah).
type CreateRequest struct {
	Amount         float64 `json:"amount" validate:"required,gte=50000"`
	IdempotencyKey string  `json:"idempotencyKey" validate:"omitempty,max=100"`
}

// Response adalah representasi withdrawal untuk pengaju/pemilik campaign & CMS.
type Response struct {
	WithdrawalID             int64      `json:"withdrawalID"`
	WithdrawalRef            string     `json:"withdrawalRef"`
	CampaignID               int64      `json:"campaignID"`
	CampaignTitle            string     `json:"campaignTitle"`
	Amount                   float64    `json:"amount"`
	Fee                      float64    `json:"fee"`
	NetAmount                float64    `json:"netAmount"`
	BeneficiaryBankCode      string     `json:"beneficiaryBankCode"`
	BeneficiaryAccountNumber string     `json:"beneficiaryAccountNumber"`
	BeneficiaryAccountHolder string     `json:"beneficiaryAccountHolder"`
	Status                   string     `json:"status"`
	RejectionReason          string     `json:"rejectionReason,omitempty"`
	ApprovedDate             *time.Time `json:"approvedDate,omitempty"`
	CompletedDate            *time.Time `json:"completedDate,omitempty"`
	CreatedDate              time.Time  `json:"createdDate"`
}

// RejectRequest adalah body admin menolak withdrawal.
type RejectRequest struct {
	Reason string `json:"reason" validate:"required,max=255"`
}

// InquiryRequest adalah body verifikasi rekening tujuan sebelum submit.
type InquiryRequest struct {
	BankCode      string `json:"bankCode" validate:"required,max=20"`
	AccountNumber string `json:"accountNumber" validate:"required,max=30"`
}

// InquiryResponse adalah hasil verifikasi rekening tujuan.
type InquiryResponse struct {
	AccountHolder string  `json:"accountHolder"`
	Fee           float64 `json:"fee"`
}

// BankListItem adalah satu entri bank tujuan transfer beserta fee live-nya.
type BankListItem struct {
	BankCode string  `json:"bankCode"`
	Name     string  `json:"name"`
	Fee      float64 `json:"fee"`
	Status   string  `json:"status"`
}

// ListFilter menampung parameter penyaringan daftar withdrawal.
type ListFilter struct {
	CampaignID        int64
	RequestedByUserID *int64
	Status            string
	Limit             int
	Offset            int
	OrderBy           string
}

// DisbursementCallbackRequest adalah body webhook disbursement dari
// Bisabiller — tidak punya field signature (proteksi via URL path secret,
// lihat withdrawal_service.ProcessCallback).
type DisbursementCallbackRequest struct {
	ReffID   string `json:"reff_id"`
	StatusID int    `json:"status_id"`
	Receipt  string `json:"receipt"`
}
