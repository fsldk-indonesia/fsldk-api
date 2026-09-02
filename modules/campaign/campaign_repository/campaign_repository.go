// Package campaign_repository adalah lapisan akses data modul campaign (GORM).
package campaign_repository

import (
	"context"
	"database/sql"
	"errors"

	"fsldk-api/modules/campaign/campaign_dto"
	"fsldk-api/modules/campaign/campaign_model"
)

// ErrNotFound dikembalikan bila campaign tidak ditemukan.
var ErrNotFound = errors.New("campaign tidak ditemukan")

// Repository adalah kontrak akses data campaign.
type Repository interface {
	List(ctx context.Context, f campaign_dto.ListFilter) ([]campaign_model.Campaign, int64, error)
	// ListLite mengembalikan seluruh campaign (id + judul saja) — dipakai
	// populasi dropdown campaign di Laporan Kantong Amal (item 6).
	ListLite(ctx context.Context) ([]campaign_model.Campaign, error)
	FindByID(ctx context.Context, id int64) (campaign_model.Campaign, error)
	FindBySlug(ctx context.Context, slug string) (campaign_model.Campaign, error)
	SlugExists(ctx context.Context, slug string, exceptID int64) (bool, error)
	CategoryExists(ctx context.Context, categoryID int64) (bool, error)
	Categories(ctx context.Context) ([]campaign_model.Category, error)
	Create(ctx context.Context, p campaign_model.CreateParams) (int64, error)
	Update(ctx context.Context, id int64, p campaign_model.UpdateParams) error
	// Delete menghapus campaign secara permanen — pemanggil (campaign_service)
	// wajib memvalidasi seluruh business-rule guard (donasi PAID belum
	// ditarik, withdrawal aktif, dsb.) sebelum memanggil ini; repository
	// tidak melakukan pengecekan apapun.
	Delete(ctx context.Context, id int64) error
	// UpdateStatus mengubah status campaign, opsional menyertakan moderationNote
	// (kosong pada transisi CRUD biasa seperti publish/pause/archive).
	UpdateStatus(ctx context.Context, id int64, status string, note sql.NullString, updatedBy int64) error
	// ReplaceImages mengganti seluruh baris ms_campaign_image milik campaign
	// dengan urls yang diberikan (delete lalu insert dalam satu transaksi) —
	// aman karena gambar pendukung murni display, tidak direferensikan tabel lain.
	ReplaceImages(ctx context.Context, campaignID int64, urls []string) error
	ListImages(ctx context.Context, campaignID int64) ([]campaign_model.Image, error)
}
