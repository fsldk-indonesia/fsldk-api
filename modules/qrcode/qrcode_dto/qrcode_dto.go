// Package qrcode_dto memuat DTO request/response modul QR Code. Seluruhnya
// murni struct data (tanpa function/method) — pemetaan model→DTO ada di service.
package qrcode_dto

// Response adalah representasi QR Code untuk API. ImageURL adalah endpoint
// publik yang mengembalikan gambar PNG QR (URL absolut), sudah menerapkan
// warna/ikon/caption yang tersimpan.
type Response struct {
	QRCodeID        int64  `json:"qrCodeID"`
	Label           string `json:"label"`
	DestinationURL  string `json:"destinationURL"`
	ForegroundColor string `json:"foregroundColor"`
	BackgroundColor string `json:"backgroundColor"`
	CenterIconURL   string `json:"centerIconURL"`
	CenterIconKey   string `json:"centerIconKey"`
	CaptionText     string `json:"captionText"`
	ImageURL        string `json:"imageURL"`
	AuthorName      string `json:"authorName"`
	CreatedDate     string `json:"createdDate"`
}

// CreateRequest adalah body membuat QR Code baru. Hanya destinationURL yang
// wajib; sisanya opsi kustomisasi gambar (warna default hitam/putih).
type CreateRequest struct {
	DestinationURL  string `json:"destinationURL" validate:"required,url,max=1000"`
	Label           string `json:"label" validate:"omitempty,max=255"`
	ForegroundColor string `json:"foregroundColor" validate:"omitempty,hexcolor"`
	BackgroundColor string `json:"backgroundColor" validate:"omitempty,hexcolor"`
	CenterIconURL   string `json:"centerIconURL" validate:"omitempty,max=40000"` // URL /uploads/... atau data URI PNG
	CenterIconKey   string `json:"centerIconKey" validate:"omitempty,max=32"`
	CaptionText     string `json:"captionText" validate:"omitempty,max=120"`
}

// UpdateRequest adalah body memperbarui QR Code (bentuk sama dengan Create).
type UpdateRequest struct {
	DestinationURL  string `json:"destinationURL" validate:"required,url,max=1000"`
	Label           string `json:"label" validate:"omitempty,max=255"`
	ForegroundColor string `json:"foregroundColor" validate:"omitempty,hexcolor"`
	BackgroundColor string `json:"backgroundColor" validate:"omitempty,hexcolor"`
	CenterIconURL   string `json:"centerIconURL" validate:"omitempty,max=40000"` // URL /uploads/... atau data URI PNG
	CenterIconKey   string `json:"centerIconKey" validate:"omitempty,max=32"`
	CaptionText     string `json:"captionText" validate:"omitempty,max=120"`
}

// ListFilter menampung parameter penyaringan daftar QR Code.
type ListFilter struct {
	Search  string
	Limit   int
	Offset  int
	OrderBy string
}
