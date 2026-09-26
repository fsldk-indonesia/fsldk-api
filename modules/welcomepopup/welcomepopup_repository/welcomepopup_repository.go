// Package welcomepopup_repository adalah lapisan akses data modul
// welcomepopup (GORM).
package welcomepopup_repository

import (
	"context"

	"fsldk-api/modules/welcomepopup/welcomepopup_model"
)

// Repository adalah kontrak akses data Welcome Popup — selalu beroperasi
// pada satu baris singleton (welcomepopup_model.SingletonID).
type Repository interface {
	Get(ctx context.Context) (welcomepopup_model.WelcomePopup, error)
	Update(ctx context.Context, popup welcomepopup_model.WelcomePopup) error
}
