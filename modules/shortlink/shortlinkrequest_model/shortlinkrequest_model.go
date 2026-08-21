// Package shortlinkrequest_model memuat entitas modul shortlink request
// (alur permintaan publik + persetujuan admin di atas modul shortlink).
// Seluruhnya murni struct data (tanpa function/method).
package shortlinkrequest_model

import "time"

// ShortLinkRequest merepresentasikan satu baris ms_shortlink_request.
// ShortKey & ReviewerName bersifat read-only (hasil join ke ms_shortlink &
// ms_user), ditandai `->` agar tidak ikut tertulis saat Create/Update.
type ShortLinkRequest struct {
	ShortLinkRequestID int64      `gorm:"column:shortLinkRequestID;primaryKey"`
	RequesterName      string     `gorm:"column:requesterName"`
	RequesterEmail     string     `gorm:"column:requesterEmail"`
	RequesterWhatsapp  string     `gorm:"column:requesterWhatsapp"`
	DestinationURL     string     `gorm:"column:destinationURL"`
	RequestedKey       *string    `gorm:"column:requestedKey"`
	Note               *string    `gorm:"column:note"`
	Status             string     `gorm:"column:status"`
	ShortLinkID        *int64     `gorm:"column:shortLinkID"`
	ShortKey           string     `gorm:"column:shortKey;->"`
	RejectionReason    *string    `gorm:"column:rejectionReason"`
	ReviewedBy         *int64     `gorm:"column:reviewedBy"` // NULL kalau ReviewedVia=ReviewedViaWhatsApp (§1a.5)
	ReviewedVia        string     `gorm:"column:reviewedVia"`
	ReviewerName       string     `gorm:"column:reviewerName;->"`
	ReviewedDate       *time.Time `gorm:"column:reviewedDate"`
	CreatedDate        time.Time  `gorm:"column:createdDate"`
}

// Status yang mungkin dimiliki sebuah permintaan.
const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
)

// Jalur yang menyelesaikan sebuah permintaan (§1a.5 techspec) — dua jalur
// approval yang saling kenal lewat mekanisme atomik yang sama (§6 techspec),
// bukan dua jalur paralel yang tidak saling kunci seperti referensi Laravel.
const (
	ReviewedViaCMS      = "cms"
	ReviewedViaWhatsApp = "whatsapp"
)
