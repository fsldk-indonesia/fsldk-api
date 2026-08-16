// Package organization_dto memuat DTO request/response modul organization.
package organization_dto

import "time"

// Response adalah representasi organisasi untuk API.
type Response struct {
	OrganizationID         int64     `json:"organizationID"`
	OrganizationTypeCode   string    `json:"organizationTypeCode"`
	OrganizationName       string    `json:"organizationName"`
	OrganizationCode       string    `json:"organizationCode"`
	ParentOrganizationID   *int64    `json:"parentOrganizationID"`
	ParentOrganizationName string    `json:"parentOrganizationName,omitempty"`
	ProvinceName           string    `json:"provinceName,omitempty"`
	CityName               string    `json:"cityName,omitempty"`
	ContactEmail           string    `json:"contactEmail,omitempty"`
	ContactPhone           string    `json:"contactPhone,omitempty"`
	IsActive               bool      `json:"isActive"`
	CreatedDate            time.Time `json:"createdDate"`
}

// MeOrganization adalah entri ringkas untuk daftar organisasi yang dapat
// diakses pengguna (dashboard switcher).
type MeOrganization struct {
	OrganizationID       int64  `json:"organizationID"`
	OrganizationTypeCode string `json:"organizationTypeCode"`
	OrganizationName     string `json:"organizationName"`
	ParentOrganizationID *int64 `json:"parentOrganizationID"`
}

// CreateRequest adalah body membuat organisasi baru. ParentOrganizationID
// wajib diisi bila membuat LDK (kecuali pembuat adalah Puskomda — parent
// otomatis dikunci ke organisasi pembuat, nilai dari client diabaikan);
// diabaikan bila membuat PUSKOMDA (parent otomatis root Puskomnas).
type CreateRequest struct {
	OrganizationTypeCode string `json:"organizationTypeCode" validate:"required,oneof=LDK PUSKOMDA"`
	OrganizationName     string `json:"organizationName" validate:"required,min=3,max=200"`
	OrganizationCode     string `json:"organizationCode" validate:"required,min=2,max=50"`
	ParentOrganizationID *int64 `json:"parentOrganizationID"`
	ProvinceName         string `json:"provinceName" validate:"max=100"`
	CityName             string `json:"cityName" validate:"max=100"`
	ContactEmail         string `json:"contactEmail" validate:"omitempty,email,max=100"`
	ContactPhone         string `json:"contactPhone" validate:"max=30"`
}

// UpdateRequest adalah body memperbarui profil organisasi.
type UpdateRequest struct {
	OrganizationName string `json:"organizationName" validate:"required,min=3,max=200"`
	ProvinceName     string `json:"provinceName" validate:"max=100"`
	CityName         string `json:"cityName" validate:"max=100"`
	ContactEmail     string `json:"contactEmail" validate:"omitempty,email,max=100"`
	ContactPhone     string `json:"contactPhone" validate:"max=30"`
}

// ListFilter menampung parameter penyaringan daftar organisasi.
type ListFilter struct {
	OrganizationTypeCode string
	Search               string
	Limit                int
	Offset               int
	OrderBy              string
}
