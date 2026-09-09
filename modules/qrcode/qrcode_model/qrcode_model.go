// Package qrcode_model memuat entitas modul QR Code. Seluruhnya murni struct
// data (tanpa function/method) — logika pemetaan berada di lapisan service.
package qrcode_model

import (
	"database/sql"
	"time"
)

// QRCode merepresentasikan satu baris ms_qrcode. Gambar QR meng-encode
// DestinationURL langsung. Field AuthorName bersifat read-only (hasil join ke
// ms_user), ditandai `->` agar tidak ikut tertulis saat Create/Update.
type QRCode struct {
	QRCodeID        int64          `gorm:"column:qrCodeID;primaryKey"`
	Label           sql.NullString `gorm:"column:label"`
	DestinationURL  string         `gorm:"column:destinationURL"`
	ForegroundColor string         `gorm:"column:foregroundColor"`
	BackgroundColor string         `gorm:"column:backgroundColor"`
	CenterIconURL   sql.NullString `gorm:"column:centerIconURL"`
	CenterIconKey   sql.NullString `gorm:"column:centerIconKey"`
	CaptionText     sql.NullString `gorm:"column:captionText"`
	CreatedBy       sql.NullInt64  `gorm:"column:createdBy"`
	AuthorName      string         `gorm:"column:authorName;->"`
	CreatedDate     time.Time      `gorm:"column:createdDate"`
	UpdatedBy       sql.NullInt64  `gorm:"column:updatedBy"`
	UpdatedDate     sql.NullTime   `gorm:"column:updatedDate"`
}
