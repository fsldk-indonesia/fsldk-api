// Package dashboard_dto memuat DTO response modul dashboard.
package dashboard_dto

// Summary adalah ringkasan angka pada dashboard.
type Summary struct {
	TotalNews     int `json:"totalNews"`
	PublishedNews int `json:"publishedNews"`
	DraftNews     int `json:"draftNews"`
	TotalUsers    int `json:"totalUsers"`
}

// RecentNews adalah ringkasan berita terbaru.
type RecentNews struct {
	NewsID      int64  `gorm:"column:newsID" json:"newsID"`
	NewsTitle   string `gorm:"column:newsTitle" json:"newsTitle"`
	IsPublished bool   `gorm:"column:isPublished" json:"isPublished"`
}
