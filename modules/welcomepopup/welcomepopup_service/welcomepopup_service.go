// Package welcomepopup_service memuat logika bisnis modul welcomepopup.
package welcomepopup_service

import (
	"context"

	"fsldk-api/modules/welcomepopup/welcomepopup_dto"
)

// Service adalah kontrak logika bisnis Welcome Popup.
type Service interface {
	Get(ctx context.Context) (welcomepopup_dto.Response, error)
	// GetPublic mengembalikan HANYA 4 field konten (welcomepopup_dto.PublicResponse) —
	// dipakai endpoint tanpa-auth, tidak pernah membawa metadata audit.
	// Mengembalikan isEnabled=false apa adanya (tidak difilter di sini) —
	// pemanggil publik (frontend) yang memutuskan render atau tidak.
	GetPublic(ctx context.Context) (welcomepopup_dto.PublicResponse, error)
	Update(ctx context.Context, req welcomepopup_dto.UpdateRequest, actorID int64) (welcomepopup_dto.Response, error)
}
