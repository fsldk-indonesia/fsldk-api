// Package qrcoderequest_model memuat entitas modul QR Code request (alur
// permintaan publik + persetujuan admin di atas modul QR Code). Seluruhnya
// murni struct data (tanpa function/method).
package qrcoderequest_model

import (
	"database/sql"
	"time"
)

// QRCodeRequest merepresentasikan satu baris ms_qrcode_request. ReviewerName
// bersifat read-only (hasil join ke ms_user), ditandai `->` agar tidak ikut
// tertulis saat Create/Update. Pemohon menyebut URL tujuan + memilih
// kustomisasi gambar (warna/ikon/caption) yang disalin ke ms_qrcode saat approve.
type QRCodeRequest struct {
	QRCodeRequestID   int64          `gorm:"column:qrCodeRequestID;primaryKey"`
	RequesterName     string         `gorm:"column:requesterName"`
	RequesterEmail    string         `gorm:"column:requesterEmail"`
	RequesterWhatsapp string         `gorm:"column:requesterWhatsapp"`
	DestinationURL    string         `gorm:"column:destinationURL"`
	ForegroundColor   string         `gorm:"column:foregroundColor"`
	BackgroundColor   string         `gorm:"column:backgroundColor"`
	CenterIconURL     sql.NullString `gorm:"column:centerIconURL"`
	CenterIconKey     sql.NullString `gorm:"column:centerIconKey"`
	CaptionText       sql.NullString `gorm:"column:captionText"`
	Note              *string        `gorm:"column:note"`
	Status            string         `gorm:"column:status"`
	QRCodeID          *int64         `gorm:"column:qrCodeID"`
	RejectionReason   *string        `gorm:"column:rejectionReason"`
	ReviewedBy        *int64         `gorm:"column:reviewedBy"` // NULL kalau ReviewedVia=ReviewedViaWhatsApp
	ReviewedVia       string         `gorm:"column:reviewedVia"`
	ReviewerName      string         `gorm:"column:reviewerName;->"`
	ReviewedDate      *time.Time     `gorm:"column:reviewedDate"`
	CreatedDate       time.Time      `gorm:"column:createdDate"`
}

// Status yang mungkin dimiliki sebuah permintaan.
const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
)

// Jalur yang menyelesaikan sebuah permintaan — dua jalur approval yang saling
// kenal lewat mekanisme atomik kondisional yang sama (WHERE status='pending').
const (
	ReviewedViaCMS      = "cms"
	ReviewedViaWhatsApp = "whatsapp"
)
