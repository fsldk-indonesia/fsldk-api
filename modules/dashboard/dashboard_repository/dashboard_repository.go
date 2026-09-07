// Package dashboard_repository adalah lapisan akses data modul dashboard.
package dashboard_repository

import (
	"context"

	"fsldk-api/modules/dashboard/dashboard_dto"
)

// Repository adalah kontrak akses data dashboard. Setiap tier memiliki bentuk
// agregasi berbeda (Section 27) sehingga dipisah per method, bukan satu query
// generik.
type Repository interface {
	LDKSummary(ctx context.Context, organizationID int64, levelisasiFormID int64) (dashboard_dto.LDKSummary, error)
	// StatusBuckets merekap jumlah LDK per bucket status Levelisasi.
	// parentOrganizationID nil berarti seluruh LDK nasional (Puskomnas).
	StatusBuckets(ctx context.Context, levelisasiFormID int64, parentOrganizationID *int64) (dashboard_dto.StatusCounts, error)
	CountLDK(ctx context.Context, parentOrganizationID *int64) (int, error)
	CountActiveKader(ctx context.Context, parentOrganizationID *int64) (int, error)
	CountPuskomda(ctx context.Context) (int, error)
	CountLevelEstablished(ctx context.Context, parentOrganizationID *int64) (int, error)
	LevelDistribution(ctx context.Context) ([]dashboard_dto.LevelCount, error)
	PerPuskomdaBreakdown(ctx context.Context) ([]dashboard_dto.PuskomdaBreakdown, error)

	// Metrik CMS Utama (Section: dashboard poin 5 miss-development-prompt-3.md).
	// Satu method per modul sidebar CMS Utama supaya widget statistik dashboard
	// mencakup semua modul yang ada (lihat komentar dashboard_dto.UtamaSummary).
	CountUsers(ctx context.Context) (int, error)
	CountRoles(ctx context.Context) (int, error)
	CountNews(ctx context.Context) (int, error)
	CountArticles(ctx context.Context) (int, error)
	CountEvents(ctx context.Context) (int, error)
	CountSchedules(ctx context.Context) (int, error)
	CountGalleries(ctx context.Context) (int, error)
	CountStructures(ctx context.Context) (int, error)
	CountCatalogBooks(ctx context.Context) (int, error)
	CountDynamicForms(ctx context.Context) (int, error)
	CountGoodsProducts(ctx context.Context) (int, error)
	CountFinanceFormats(ctx context.Context) (int, error)
	CountCampaigns(ctx context.Context) (int, error)
	SumDonationCollected(ctx context.Context) (float64, error)
	CountComments(ctx context.Context) (int, error)
	CountShortlinks(ctx context.Context) (int, error)
	CountActiveSubscribers(ctx context.Context) (int, error)
	CountUnreadContactMessages(ctx context.Context) (int, error)
	CountPendingJobs(ctx context.Context) (int, error)
}
