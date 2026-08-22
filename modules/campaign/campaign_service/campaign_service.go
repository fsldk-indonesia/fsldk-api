// Package campaign_service memuat logika bisnis modul campaign.
package campaign_service

import (
	"context"

	"fsldk-api/base/dto"
	"fsldk-api/modules/campaign/campaign_dto"
)

// CallerScope menampung identitas & scope organisasi pengguna pemanggil,
// diambil dari klaim token (lihat base/appctx).
type CallerScope struct {
	UserID               int64
	OrganizationID       *int64
	OrganizationTypeCode string
	WildcardTierAccess   string
}

// OrgAccessChecker adalah irisan sempit dari organization_service.Service
// yang dibutuhkan modul ini — menerima interface (bukan mengimpor
// organization_service langsung) menghindari dependency package yang keras,
// mengikuti pola CommentCleaner di modules/news.
type OrgAccessChecker interface {
	IsAccessible(ctx context.Context, callerOrganizationID *int64, callerOrganizationTypeCode, wildcardTierAccess string, targetOrganizationID int64) (bool, error)
}

// Service adalah kontrak logika bisnis campaign.
type Service interface {
	PublicList(ctx context.Context, q dto.ListQuery, categoryID int64) ([]campaign_dto.Response, int, error)
	PublicDetail(ctx context.Context, slug string) (campaign_dto.DetailResponse, error)
	Categories(ctx context.Context) ([]campaign_dto.CategoryResponse, error)

	MyList(ctx context.Context, caller CallerScope, q dto.ListQuery) ([]campaign_dto.Response, int, error)
	Create(ctx context.Context, caller CallerScope, req campaign_dto.CreateRequest) (campaign_dto.DetailResponse, error)
	Update(ctx context.Context, id int64, caller CallerScope, req campaign_dto.UpdateRequest) (campaign_dto.DetailResponse, error)
	Submit(ctx context.Context, id int64, caller CallerScope) (campaign_dto.DetailResponse, error)

	CMSList(ctx context.Context, q dto.ListQuery, status string, categoryID int64) ([]campaign_dto.Response, int, error)
	Get(ctx context.Context, id int64) (campaign_dto.DetailResponse, error)
	ReviewHistory(ctx context.Context, id int64) ([]campaign_dto.ReviewResponse, error)
	Review(ctx context.Context, id int64, caller CallerScope, req campaign_dto.ReviewRequest) (campaign_dto.DetailResponse, error)
	Publish(ctx context.Context, id int64, caller CallerScope) (campaign_dto.DetailResponse, error)
	Pause(ctx context.Context, id int64, caller CallerScope) (campaign_dto.DetailResponse, error)
	Resume(ctx context.Context, id int64, caller CallerScope) (campaign_dto.DetailResponse, error)
	Archive(ctx context.Context, id int64, caller CallerScope) (campaign_dto.DetailResponse, error)
}
