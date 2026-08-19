// Package organization_repository adalah lapisan akses data modul organization.
package organization_repository

import (
	"context"
	"errors"

	"fsldk-api/modules/organization/organization_dto"
	"fsldk-api/modules/organization/organization_model"
)

// ErrNotFound dikembalikan bila organisasi tidak ditemukan.
var ErrNotFound = errors.New("organisasi tidak ditemukan")

// Repository adalah kontrak akses data organisasi.
type Repository interface {
	FindByID(ctx context.Context, id int64) (organization_model.Organization, error)
	ExistsByCode(ctx context.Context, code string) (bool, error)
	TypeCodeByID(ctx context.Context, id int64) (string, error)
	IsChildOf(ctx context.Context, targetID, parentID int64) (bool, error)

	// ListAll mengembalikan seluruh organisasi (dipakai caller bertipe PUSKOMNAS/wildcard root).
	ListAll(ctx context.Context, f organization_dto.ListFilter) ([]organization_model.Organization, int64, error)
	// ListByTypes membatasi hasil pada organizationTypeCode tertentu (dipakai wildcardTierAccess).
	ListByTypes(ctx context.Context, types []string, f organization_dto.ListFilter) ([]organization_model.Organization, int64, error)
	// ListSelfAndChildren mengembalikan organisasi itu sendiri beserta anak langsungnya (dipakai caller PUSKOMDA).
	ListSelfAndChildren(ctx context.Context, id int64, f organization_dto.ListFilter) ([]organization_model.Organization, int64, error)
	Children(ctx context.Context, id int64) ([]organization_model.Organization, error)
	RootPuskomnasID(ctx context.Context) (int64, error)
	// ListActiveByType mengembalikan seluruh organisasi aktif bertipe
	// tertentu tanpa cakupan akses (dipakai direktori pemilihan bebas).
	ListActiveByType(ctx context.Context, typeCode string) ([]organization_model.Organization, error)

	Create(ctx context.Context, p organization_model.CreateParams) (int64, error)
	Update(ctx context.Context, id int64, name, province, city, email, phone string, updatedBy int64) error
	SetActive(ctx context.Context, id int64, active bool, updatedBy int64) error
}
