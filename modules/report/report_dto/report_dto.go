// Package report_dto memuat DTO modul report.
package report_dto

import "database/sql"

// SubmissionRow adalah satu baris laporan submission untuk ekspor.
type SubmissionRow struct {
	OrganizationName string         `gorm:"column:organizationName"`
	ProvinceName     sql.NullString `gorm:"column:provinceName"`
	CityName         sql.NullString `gorm:"column:cityName"`
	Status           string         `gorm:"column:status"`
	LevelLabel       sql.NullString `gorm:"column:levelLabel"`
	SubmittedDate    sql.NullTime   `gorm:"column:submittedDate"`
	LastUpdatedDate  sql.NullTime   `gorm:"column:lastUpdatedDate"`
}

// ExportFilter menampung parameter penyaringan ekspor laporan submission.
type ExportFilter struct {
	FormCode string
	Status   string
	Format   string // "xlsx" atau "csv"
}

// ExportResult adalah berkas hasil ekspor siap dikirim sebagai response.
type ExportResult struct {
	FileName    string
	ContentType string
	Data        []byte
}
