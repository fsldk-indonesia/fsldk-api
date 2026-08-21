// Package organization_model memuat entitas modul organization. Murni struct data.
package organization_model

import (
	"database/sql"
	"time"
)

// Organization merepresentasikan satu baris ms_organization. ParentOrganizationName
// bersifat read-only (hasil join), ditandai `->` agar tidak ikut tertulis saat Create/Update.
type Organization struct {
	OrganizationID         int64          `gorm:"column:organizationID;primaryKey"`
	OrganizationTypeCode   string         `gorm:"column:organizationTypeCode"`
	ParentOrganizationID   sql.NullInt64  `gorm:"column:parentOrganizationID"`
	ParentOrganizationName sql.NullString `gorm:"column:parentOrganizationName;->"`
	OrganizationName       string         `gorm:"column:organizationName"`
	OrganizationCode       string         `gorm:"column:organizationCode"`
	ProvinceName           sql.NullString `gorm:"column:provinceName"`
	CityName               sql.NullString `gorm:"column:cityName"`
	ContactEmail           sql.NullString `gorm:"column:contactEmail"`
	ContactPhone           sql.NullString `gorm:"column:contactPhone"`
	IsActive               bool           `gorm:"column:isActive"`
	PhotoURL               sql.NullString `gorm:"column:photoURL"`
	CreatedDate            time.Time      `gorm:"column:createdDate"`
}

// CreateParams menampung data untuk membuat organisasi baru.
type CreateParams struct {
	OrganizationTypeCode string
	ParentOrganizationID sql.NullInt64
	OrganizationName     string
	OrganizationCode     string
	ProvinceName         sql.NullString
	CityName             sql.NullString
	ContactEmail         sql.NullString
	ContactPhone         sql.NullString
	CreatedBy            sql.NullInt64
}
