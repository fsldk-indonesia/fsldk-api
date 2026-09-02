// Package withdrawal_dto memuat DTO request/response modul withdrawal.
// Murni struct data.
package withdrawal_dto

import "time"

// CreateRequest adalah body mengajukan penarikan saldo baru. Rekening
// tujuan diinput ulang setiap pengajuan (revisi 2026-09-01 — beneficiary
// TIDAK LAGI tersimpan di campaign, lihat campaign_dto) dan divalidasi ulang
// via inquiry live gateway di withdrawal_service.Request; pemilik rekening
// (beneficiaryAccountHolder) diambil dari hasil inquiry yang terverifikasi
// gateway, bukan dari input client, supaya tidak bisa dipalsukan.
type CreateRequest struct {
	Amount                   float64 `json:"amount" validate:"required,gte=50000"`
	BeneficiaryBankCode      string  `json:"beneficiaryBankCode" validate:"required,max=20"`
	BeneficiaryAccountNumber string  `json:"beneficiaryAccountNumber" validate:"required,max=30"`
	IdempotencyKey           string  `json:"idempotencyKey" validate:"omitempty,max=100"`
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

// SecurityVerifyRequest adalah body submit verifikasi keamanan withdrawal.
// OtpCode hanya wajib diisi bila withdrawal terkena trigger risk-based
// (lihat withdrawal_service.VerifySecurity) — untuk withdrawal rutin,
// Password saja sudah cukup (Option B).
type SecurityVerifyRequest struct {
	Password string `json:"password" validate:"required"`
	OtpCode  string `json:"otpCode" validate:"omitempty,len=6"`
}

// DisbursementCallbackRequest adalah body webhook disbursement dari
// Bisabiller — tidak punya field signature (proteksi via URL path secret,
// lihat withdrawal_service.ProcessCallback).
type DisbursementCallbackRequest struct {
	ReffID   string `json:"reff_id"`
	StatusID int    `json:"status_id"`
	Receipt  string `json:"receipt"`
}
