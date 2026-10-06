// Package rapimnas_repository adalah lapisan akses data modul rapimnas (GORM).
package rapimnas_repository

import (
	"context"

	"fsldk-api/modules/rapimnas/rapimnas_model"
)

// Repository adalah kontrak akses data Rapimnas — selalu beroperasi pada
// satu Aggregate (baris singleton ms_rapimnas_setting + 6 child collection).
type Repository interface {
	// Get mengambil baris singleton beserta seluruh child collection, masing-
	// masing diurutkan sortOrder ASC.
	Get(ctx context.Context) (*rapimnas_model.Aggregate, error)
	// Save menjalankan satu transaksi: upsert baris singleton, lalu untuk
	// setiap 6 tabel anak, hapus seluruh baris lama dan buat ulang baris yang
	// disubmit ("replace whole collection on save").
	Save(ctx context.Context, agg *rapimnas_model.Aggregate) error
}
