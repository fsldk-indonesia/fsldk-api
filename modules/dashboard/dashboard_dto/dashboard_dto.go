// Package dashboard_dto memuat DTO response modul dashboard.
package dashboard_dto

// Summary adalah ringkasan angka pada dashboard.
type Summary struct {
	TotalNews     int `json:"totalNews"`
	TotalArticles int `json:"totalArticles"`
	TotalUsers    int `json:"totalUsers"`
}
