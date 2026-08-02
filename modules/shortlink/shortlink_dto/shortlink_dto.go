// Package shortlink_dto memuat DTO request/response modul shortlink.
// Seluruhnya murni struct data (tanpa function/method) — pemetaan
// model→DTO berada di service.
package shortlink_dto

// Response adalah representasi shortlink untuk API.
type Response struct {
	ShortLinkID    int64  `json:"shortLinkID"`
	ShortKey       string `json:"shortKey"`
	DestinationURL string `json:"destinationURL"`
	ShortURL       string `json:"shortURL"`
	VisitCount     int64  `json:"visitCount"`
	AuthorName     string `json:"authorName"`
	CreatedDate    string `json:"createdDate"`
}

// CreateRequest adalah body membuat shortlink baru. ShortKey opsional — bila
// kosong, kunci acak akan dibuatkan otomatis oleh service. Huruf, angka, dan
// tanda hubung (-) diperbolehkan.
type CreateRequest struct {
	DestinationURL string `json:"destinationURL" validate:"required,url,max=1000"`
	ShortKey       string `json:"shortKey" validate:"omitempty,shortlinkkey,min=3,max=30"`
}

// UpdateRequest adalah body memperbarui shortlink.
type UpdateRequest struct {
	DestinationURL string `json:"destinationURL" validate:"required,url,max=1000"`
	ShortKey       string `json:"shortKey" validate:"required,shortlinkkey,min=3,max=30"`
}

// ResolveResponse adalah hasil resolusi publik sebuah kunci shortlink,
// dikonsumsi frontend fsldk-web untuk redirect di sisi browser.
type ResolveResponse struct {
	DestinationURL string `json:"destinationURL"`
}

// ListFilter menampung parameter penyaringan daftar shortlink.
type ListFilter struct {
	Search  string
	Limit   int
	Offset  int
	OrderBy string
}
