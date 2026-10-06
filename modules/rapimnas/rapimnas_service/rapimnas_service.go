// Package rapimnas_service memuat logika bisnis modul rapimnas.
package rapimnas_service

import (
	"context"

	"fsldk-api/modules/rapimnas/rapimnas_dto"
)

// Service adalah kontrak logika bisnis Rapimnas.
type Service interface {
	// Get mengembalikan payload CMS lengkap (termasuk Resources yang
	// IsVisible=false, dan metadata audit).
	Get(ctx context.Context) (rapimnas_dto.CMSResponse, error)
	// GetPublic mengembalikan payload publik — Resources hanya yang
	// IsVisible=true, tanpa metadata audit. Dipakai endpoint tanpa-auth.
	GetPublic(ctx context.Context) (rapimnas_dto.PublicResponse, error)
	// Update memvalidasi & menyimpan seluruh payload nested sekali jalan
	// (satu tombol Simpan), lalu mengembalikan payload CMS terbaru.
	Update(ctx context.Context, req rapimnas_dto.UpdateRequest, actorID int64) (rapimnas_dto.CMSResponse, error)
}
