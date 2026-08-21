// Package shortlinkrequest_dto memuat DTO request/response modul shortlink
// request. Seluruhnya murni struct data (tanpa function/method).
package shortlinkrequest_dto

// SubmitRequest adalah body permintaan shortlink publik. Seluruh field wajib
// diisi (mengikuti perilaku form referensi) — TERMASUK RequestedKey & Note,
// beda dari desain awal techspec §5/§6 yang membiarkan keduanya opsional
// (kunci acak dibuatkan otomatis saat approve bila kosong). Fallback
// generate-key otomatis di shortlinkrequest_service.Approve tetap
// dipertahankan (bukan dihapus) untuk baris ms_shortlink_request lama yang
// requestedKey-nya NULL sebelum perubahan ini berlaku.
type SubmitRequest struct {
	RequesterName     string `json:"requesterName" validate:"required,min=2,max=255"`
	RequesterEmail    string `json:"requesterEmail" validate:"required,email,max=255"`
	RequesterWhatsapp string `json:"requesterWhatsapp" validate:"required,phonenumber,max=20"` // reuse validator existing; normalisasi ke 62xxxx di service
	DestinationURL    string `json:"destinationURL" validate:"required,url,max=1000"`
	RequestedKey      string `json:"requestedKey" validate:"required,shortlinkkey,min=3,max=30"` // reuse validator existing
	Note              string `json:"note" validate:"required,max=1000"`
}

// RejectRequest adalah body penolakan permintaan.
type RejectRequest struct {
	RejectionReason string `json:"rejectionReason" validate:"required,max=500"`
}

// Response adalah representasi permintaan shortlink untuk API.
type Response struct {
	ShortLinkRequestID int64  `json:"shortLinkRequestID"`
	RequesterName      string `json:"requesterName"`
	RequesterEmail     string `json:"requesterEmail"`
	RequesterWhatsapp  string `json:"requesterWhatsapp"`
	DestinationURL     string `json:"destinationURL"`
	RequestedKey       string `json:"requestedKey"`
	Note               string `json:"note"`
	Status             string `json:"status"`
	ShortLinkID        int64  `json:"shortLinkID"`
	ShortKey           string `json:"shortKey"`
	ShortURL           string `json:"shortURL"`
	RejectionReason    string `json:"rejectionReason"`
	ReviewedVia        string `json:"reviewedVia"` // "cms" | "whatsapp" (§1a.5)
	ReviewerName       string `json:"reviewerName"`
	ReviewedDate       string `json:"reviewedDate"`
	CreatedDate        string `json:"createdDate"`
}

// PICResponse adalah info kontak PIC (Penanggung Jawab) shortlink untuk
// ditampilkan di halaman publik pengajuan (kartu "Konfirmasi via WhatsApp")
// — subset read-only dari ms_setting, BUKAN endpoint /settings penuh yang
// tetap CMS-only (§1a.3 techspec). PICWhatsapp bisa kosong bila App Settings
// belum dikonfigurasi — bukan error, frontend cukup menyembunyikan kartunya.
type PICResponse struct {
	PICName     string `json:"picName"`
	PICWhatsapp string `json:"picWhatsapp"`
}

// ListFilter menampung parameter penyaringan daftar permintaan shortlink.
type ListFilter struct {
	Status  string // pending|approved|rejected|"" (semua)
	Search  string
	Limit   int
	Offset  int
	OrderBy string
}
