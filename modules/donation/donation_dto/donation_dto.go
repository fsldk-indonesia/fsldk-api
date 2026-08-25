// Package donation_dto memuat DTO request/response modul donation. Murni struct data.
package donation_dto

import "time"

// CreateRequest adalah body membuat donasi baru ke sebuah campaign
// (selalu berstatus PENDING — belum ada integrasi gateway pembayaran di
// fase ini, lihat donation_service).
type CreateRequest struct {
	Amount          float64 `json:"amount" validate:"required,gte=1000"`
	DonorName       string  `json:"donorName" validate:"required,max=150"`
	DonorEmail      string  `json:"donorEmail" validate:"required,email,max=150"`
	DonorPhone      string  `json:"donorPhone" validate:"required,max=30"`
	DonorAge        string  `json:"donorAge" validate:"max=10"`
	DonorDomicile   string  `json:"donorDomicile" validate:"max=150"`
	DonorOccupation string  `json:"donorOccupation" validate:"max=150"`
	IsAnonymous     bool    `json:"isAnonymous"`
	Message         string  `json:"message" validate:"max=1000"`
	IdempotencyKey  string  `json:"idempotencyKey" validate:"omitempty,max=100"`
}

// Response adalah representasi donasi untuk pembuat/pemilik donasi sendiri
// (halaman konfirmasi/status pembayaran) — donorEmail/donorPhone sengaja
// tidak diikutsertakan (klien sudah punya nilai itu dari form yang baru
// dikirim, tidak perlu di-echo balik lewat link publicRef yang bisa saja
// dibagikan).
type Response struct {
	DonationID    int64      `json:"donationID"`
	PublicRef     string     `json:"publicRef"`
	CampaignID    int64      `json:"campaignID"`
	CampaignTitle string     `json:"campaignTitle"`
	CampaignSlug  string     `json:"campaignSlug"`
	DonorName     string     `json:"donorName"`
	IsAnonymous   bool       `json:"isAnonymous"`
	Message       string     `json:"message,omitempty"`
	Amount        float64    `json:"amount"`
	AdminFee      float64    `json:"adminFee"`
	TotalAmount   float64    `json:"totalAmount"`
	PaymentStatus string     `json:"paymentStatus"`
	Gateway       string     `json:"gateway"`
	QrPayload     string     `json:"qrPayload,omitempty"`
	PaymentCode   string     `json:"paymentCode,omitempty"`
	PaymentLink   string     `json:"paymentLink,omitempty"`
	ExpiredDate   *time.Time `json:"expiredDate,omitempty"`
	CreatedDate   time.Time  `json:"createdDate"`
}

// StatusResponse adalah hasil ringan endpoint polling status pembayaran.
type StatusResponse struct {
	PaymentStatus string `json:"paymentStatus"`
}

// CallbackRequest adalah body webhook payment callback dari Bisabiller.
// TransactionTotal bertipe string karena begitu apa adanya dari payload
// gateway (dikonversi ke angka di donation_service).
type CallbackRequest struct {
	TransactionID    string `json:"transaction_id"`
	TransactionTotal string `json:"transaction_total"`
	Signature        string `json:"signature"`
	StatusID         int    `json:"status_id"`
}

// ListFilter menampung parameter penyaringan daftar donasi.
type ListFilter struct {
	CampaignID  int64
	DonorUserID *int64
	Status      string
	Limit       int
	Offset      int
	OrderBy     string
}

// PublicDonationItem adalah satu baris "donatur terbaru" pada halaman
// campaign detail publik — hanya field yang aman ditampilkan ke pengunjung
// (donorName sudah dimasking bila isAnonymous, tanpa email/telepon/fee).
type PublicDonationItem struct {
	DonorName   string    `json:"donorName"`
	IsAnonymous bool      `json:"isAnonymous"`
	Amount      float64   `json:"amount"`
	Message     string    `json:"message,omitempty"`
	CreatedDate time.Time `json:"createdDate"`
}
