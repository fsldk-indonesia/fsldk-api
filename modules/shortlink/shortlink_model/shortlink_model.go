// Package shortlink_model memuat entitas modul shortlink. Seluruhnya murni
// struct data (tanpa function/method) — logika pemetaan berada di lapisan service.
package shortlink_model

import (
	"database/sql"
	"time"
)

// ShortLink merepresentasikan satu baris ms_shortlink. Field AuthorName
// bersifat read-only (hasil join ke ms_user), ditandai `->` agar tidak ikut
// tertulis saat Create/Update.
type ShortLink struct {
	ShortLinkID    int64         `gorm:"column:shortLinkID;primaryKey"`
	ShortKey       string        `gorm:"column:shortKey"`
	DestinationURL string        `gorm:"column:destinationURL"`
	VisitCount     int64         `gorm:"column:visitCount"`
	CreatedBy      sql.NullInt64 `gorm:"column:createdBy"`
	AuthorName     string        `gorm:"column:authorName;->"`
	CreatedDate    time.Time     `gorm:"column:createdDate"`
	UpdatedBy      sql.NullInt64 `gorm:"column:updatedBy"`
	UpdatedDate    sql.NullTime  `gorm:"column:updatedDate"`
}
