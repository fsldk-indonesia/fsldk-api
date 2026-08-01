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
	NewsID      int64  `db:"newsID" json:"newsID"`
	NewsTitle   string `db:"newsTitle" json:"newsTitle"`
	IsPublished bool   `db:"isPublished" json:"isPublished"`
}
