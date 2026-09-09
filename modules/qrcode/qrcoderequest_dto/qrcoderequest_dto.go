// Package qrcoderequest_dto memuat DTO request/response modul QR Code request.
// Seluruhnya murni struct data (tanpa function/method).
package qrcoderequest_dto

// SubmitRequest adalah body permintaan QR Code publik. Tidak ada requestedKey
// — gambar QR meng-encode destinationURL langsung. Pemohon boleh ikut memilih
// kustomisasi gambar (warna/ikon/caption) yang disalin ke ms_qrcode saat approve.
type SubmitRequest struct {
	RequesterName     string `json:"requesterName" validate:"required,min=2,max=255"`
	RequesterEmail    string `json:"requesterEmail" validate:"required,email,max=255"`
	RequesterWhatsapp string `json:"requesterWhatsapp" validate:"required,phonenumber,max=20"` // dinormalisasi ke 62xxxx di service
	DestinationURL    string `json:"destinationURL" validate:"required,url,max=1000"`
	ForegroundColor   string `json:"foregroundColor" validate:"omitempty,hexcolor"`
	BackgroundColor   string `json:"backgroundColor" validate:"omitempty,hexcolor"`
	CenterIconURL     string `json:"centerIconURL" validate:"omitempty,max=40000"`
	CenterIconKey     string `json:"centerIconKey" validate:"omitempty,max=32"`
	CaptionText       string `json:"captionText" validate:"omitempty,max=120"`
	Note              string `json:"note" validate:"required,max=1000"`
}

// RejectRequest adalah body penolakan permintaan.
type RejectRequest struct {
	RejectionReason string `json:"rejectionReason" validate:"required,max=500"`
}

// Response adalah representasi permintaan QR Code untuk API.
type Response struct {
	QRCodeRequestID   int64  `json:"qrCodeRequestID"`
	RequesterName     string `json:"requesterName"`
	RequesterEmail    string `json:"requesterEmail"`
	RequesterWhatsapp string `json:"requesterWhatsapp"`
	DestinationURL    string `json:"destinationURL"`
	ForegroundColor   string `json:"foregroundColor"`
	BackgroundColor   string `json:"backgroundColor"`
	CenterIconURL     string `json:"centerIconURL"`
	CenterIconKey     string `json:"centerIconKey"`
	CaptionText       string `json:"captionText"`
	Note              string `json:"note"`
	Status            string `json:"status"`
	QRCodeID          int64  `json:"qrCodeID"`
	ImageURL          string `json:"imageURL"` // terisi hanya setelah approved
	RejectionReason   string `json:"rejectionReason"`
	ReviewedVia       string `json:"reviewedVia"` // "cms" | "whatsapp"
	ReviewerName      string `json:"reviewerName"`
	ReviewedDate      string `json:"reviewedDate"`
	CreatedDate       string `json:"createdDate"`
}

// PICResponse adalah info kontak PIC (Penanggung Jawab) QR Code untuk
// ditampilkan di halaman publik pengajuan (kartu "Konfirmasi via WhatsApp").
type PICResponse struct {
	PICName     string `json:"picName"`
	PICWhatsapp string `json:"picWhatsapp"`
}

// ListFilter menampung parameter penyaringan daftar permintaan QR Code.
type ListFilter struct {
	Status  string // pending|approved|rejected|"" (semua)
	Search  string
	Limit   int
	Offset  int
	OrderBy string
}
