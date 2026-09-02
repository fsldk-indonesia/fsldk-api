package campaign_service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/idgen"
	"fsldk-api/base/slug"
	"fsldk-api/constants"
	"fsldk-api/modules/campaign/campaign_dto"
	"fsldk-api/modules/campaign/campaign_model"
	"fsldk-api/modules/campaign/campaign_repository"
	"fsldk-api/modules/wallet/wallet_service"
	"fsldk-api/pkg/auditlog"
)

// FinanceAuditor adalah irisan sempit auditlog.Logger yang dibutuhkan modul
// ini — pola sama JobEnqueuer, memenuhi kontrak dengan *auditlog.Logger
// sungguhan sekaligus memudahkan fake di unit test tanpa perlu *gorm.DB.
type FinanceAuditor interface {
	LogFinance(ctx context.Context, e auditlog.Entry)
}

// DonationChecker adalah irisan sempit donation_repository.Repository yang
// dibutuhkan guard Delete() — mencegah campaign yang masih punya donasi
// aktif/belum ditarik terhapus (pola sama celengan syahid destroyAdminCampaign).
type DonationChecker interface {
	CountPaidByCampaign(ctx context.Context, campaignID int64) (int64, error)
	CountPendingByCampaign(ctx context.Context, campaignID int64) (int64, error)
}

// WithdrawalChecker adalah irisan sempit withdrawal_repository.Repository
// yang dibutuhkan guard Delete().
type WithdrawalChecker interface {
	CountNonFinalByCampaign(ctx context.Context, campaignID int64) (int64, error)
}

var sortColumns = map[string]string{
	"createdDate":          "c.createdDate",
	"title":                "c.title",
	"targetAmount":         "c.targetAmount",
	"collectedAmountCache": "c.collectedAmountCache",
}

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct {
	repo       campaign_repository.Repository
	orgAccess  OrgAccessChecker
	audit      FinanceAuditor
	donations  DonationChecker
	withdrawal WithdrawalChecker
	wallet     wallet_service.Service
}

// NewService membuat Service campaign.
func NewService(repo campaign_repository.Repository, orgAccess OrgAccessChecker, audit FinanceAuditor, donations DonationChecker, withdrawal WithdrawalChecker, wallet wallet_service.Service) Service {
	return &ServiceImpl{repo: repo, orgAccess: orgAccess, audit: audit, donations: donations, withdrawal: withdrawal, wallet: wallet}
}

// ---------- Public ----------

func (s *ServiceImpl) PublicList(ctx context.Context, q dto.ListQuery, categoryID int64) ([]campaign_dto.Response, int, error) {
	return s.list(ctx, campaign_dto.ListFilter{
		Status:     constants.CampaignStatusPublished,
		CategoryID: categoryID,
		Search:     q.Search,
		Limit:      q.Limit,
		Offset:     q.Offset(),
		OrderBy:    q.OrderBy(sortColumns, "c.createdDate DESC"),
	})
}

func (s *ServiceImpl) PublicDetail(ctx context.Context, slugStr string) (campaign_dto.DetailResponse, error) {
	c, err := s.repo.FindBySlug(ctx, slugStr)
	if err != nil || c.Status != constants.CampaignStatusPublished {
		return campaign_dto.DetailResponse{}, apperror.NotFound("Campaign tidak ditemukan")
	}
	return s.toDetail(ctx, c)
}

func (s *ServiceImpl) Categories(ctx context.Context) ([]campaign_dto.CategoryResponse, error) {
	cats, err := s.repo.Categories(ctx)
	if err != nil {
		return nil, apperror.Internal("")
	}
	out := make([]campaign_dto.CategoryResponse, 0, len(cats))
	for _, c := range cats {
		out = append(out, campaign_dto.CategoryResponse{
			CampaignCategoryID: c.CampaignCategoryID,
			CategoryCode:       c.CategoryCode,
			CategoryName:       c.CategoryName,
		})
	}
	return out, nil
}

// ---------- CMS (create/update/delete murni permission-gated) ----------

func (s *ServiceImpl) Create(ctx context.Context, caller CallerScope, req campaign_dto.CreateRequest) (campaign_dto.DetailResponse, error) {
	if err := s.validateCategory(ctx, req.CategoryID); err != nil {
		return campaign_dto.DetailResponse{}, err
	}
	orgID, err := s.resolveOrganization(ctx, caller, req.OrganizationID)
	if err != nil {
		return campaign_dto.DetailResponse{}, err
	}
	startDate, endDate, err := parseDateRange(req.StartDate, req.EndDate)
	if err != nil {
		return campaign_dto.DetailResponse{}, err
	}
	slugStr, err := s.uniqueSlug(ctx, req.Title, 0)
	if err != nil {
		return campaign_dto.DetailResponse{}, err
	}

	id, err := s.repo.Create(ctx, campaign_model.CreateParams{
		PublicRef:                idgen.NewUUIDv4(),
		Slug:                     slugStr,
		Title:                    req.Title,
		CategoryID:               req.CategoryID,
		OrganizationID:           orgID,
		ProvinceName:             nullStringFrom(req.ProvinceName),
		CityName:                 nullStringFrom(req.CityName),
		Story:                    req.Story,
		Goals:                    req.Goals,
		CoverImageUrl:            req.CoverImageUrl,
		TargetAmount:             req.TargetAmount,
		PicName:                  req.PicName,
		PicPhone:                 req.PicPhone,
		OrganizationNameOverride: nullStringFrom(req.OrganizationNameOverride),
		OrganizationLogoUrl:      nullStringFrom(req.OrganizationLogoUrl),
		OrganizationLinkUrl:      nullStringFrom(req.OrganizationLinkUrl),
		StartDate:                startDate,
		EndDate:                  endDate,
		IsAnonymousAllowed:       boolOrDefault(req.IsAnonymousAllowed, true),
		CreatedBy:                caller.UserID,
	})
	if err != nil {
		return campaign_dto.DetailResponse{}, apperror.Internal("Gagal membuat campaign")
	}
	if len(req.SupportingImageUrls) > 0 {
		if err := s.repo.ReplaceImages(ctx, id, req.SupportingImageUrls); err != nil {
			return campaign_dto.DetailResponse{}, apperror.Internal("")
		}
	}
	s.audit.LogFinance(ctx, auditlog.Entry{
		ActorUserID: caller.UserID, Action: "campaign.created", Entity: "campaign", EntityID: id,
		After: map[string]interface{}{"title": req.Title, "targetAmount": req.TargetAmount},
	})
	return s.getDetail(ctx, id)
}

func (s *ServiceImpl) Update(ctx context.Context, id int64, caller CallerScope, req campaign_dto.UpdateRequest) (campaign_dto.DetailResponse, error) {
	camp, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return campaign_dto.DetailResponse{}, apperror.NotFound("Campaign tidak ditemukan")
	}
	if camp.Status == constants.CampaignStatusArchived {
		return campaign_dto.DetailResponse{}, apperror.InvalidStatusTransition("Campaign yang sudah diarsipkan tidak dapat diubah")
	}
	if err := s.validateCategory(ctx, req.CategoryID); err != nil {
		return campaign_dto.DetailResponse{}, err
	}
	orgID, err := s.resolveOrganization(ctx, caller, req.OrganizationID)
	if err != nil {
		return campaign_dto.DetailResponse{}, err
	}
	startDate, endDate, err := parseDateRange(req.StartDate, req.EndDate)
	if err != nil {
		return campaign_dto.DetailResponse{}, err
	}
	slugStr := camp.Slug
	if camp.Title != req.Title {
		slugStr, err = s.uniqueSlug(ctx, req.Title, id)
		if err != nil {
			return campaign_dto.DetailResponse{}, err
		}
	}

	if err := s.repo.Update(ctx, id, campaign_model.UpdateParams{
		Slug:                     slugStr,
		Title:                    req.Title,
		CategoryID:               req.CategoryID,
		OrganizationID:           orgID,
		ProvinceName:             nullStringFrom(req.ProvinceName),
		CityName:                 nullStringFrom(req.CityName),
		Story:                    req.Story,
		Goals:                    req.Goals,
		LatestUpdate:             nullStringFrom(req.LatestUpdate),
		CoverImageUrl:            req.CoverImageUrl,
		TargetAmount:             req.TargetAmount,
		PicName:                  req.PicName,
		PicPhone:                 req.PicPhone,
		OrganizationNameOverride: nullStringFrom(req.OrganizationNameOverride),
		OrganizationLogoUrl:      nullStringFrom(req.OrganizationLogoUrl),
		OrganizationLinkUrl:      nullStringFrom(req.OrganizationLinkUrl),
		StartDate:                startDate,
		EndDate:                  endDate,
		IsAnonymousAllowed:       boolOrDefault(req.IsAnonymousAllowed, camp.IsAnonymousAllowed),
		UpdatedBy:                caller.UserID,
	}); err != nil {
		return campaign_dto.DetailResponse{}, apperror.Internal("")
	}
	// nil berarti "tidak diubah"; slice kosong non-nil berarti "hapus semua
	// gambar pendukung" — lihat komentar campaign_dto.UpdateRequest.
	if req.SupportingImageUrls != nil {
		if err := s.repo.ReplaceImages(ctx, id, req.SupportingImageUrls); err != nil {
			return campaign_dto.DetailResponse{}, apperror.Internal("")
		}
	}
	s.audit.LogFinance(ctx, auditlog.Entry{
		ActorUserID: caller.UserID, Action: "campaign.updated", Entity: "campaign", EntityID: id,
		Before: map[string]interface{}{"title": camp.Title, "targetAmount": camp.TargetAmount},
		After:  map[string]interface{}{"title": req.Title, "targetAmount": req.TargetAmount},
	})
	return s.getDetail(ctx, id)
}

// Delete menghapus campaign permanen. Guard mengikuti persis
// destroyAdminCampaign celengan syahid (ldksyahid-app), minus pengecekan
// rekonsiliasi live Bisatopup (sudah tercakup di Laporan Kantong Amal —
// item 6 — sebagai pengawasan terpisah, bukan blocker hapus per-campaign):
//  1. Ada donasi PAID dengan withdrawal masih berjalan → blokir.
//  2. Ada donasi PAID dengan saldo tersedia (belum ditarik habis) → blokir.
//  3. Ada donasi PENDING aktif (gateway maupun manual) → blokir.
func (s *ServiceImpl) Delete(ctx context.Context, id int64) error {
	camp, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return apperror.NotFound("Campaign tidak ditemukan")
	}

	paidCount, err := s.donations.CountPaidByCampaign(ctx, id)
	if err != nil {
		return apperror.Internal("")
	}
	if paidCount > 0 {
		nonFinalWithdrawal, err := s.withdrawal.CountNonFinalByCampaign(ctx, id)
		if err != nil {
			return apperror.Internal("")
		}
		if nonFinalWithdrawal > 0 {
			return apperror.Unprocessable("Campaign tidak dapat dihapus: masih ada penarikan saldo yang sedang berjalan")
		}
		bal, err := s.wallet.GetBalance(ctx, id)
		if err != nil {
			return apperror.Internal("")
		}
		if bal.AvailableBalance > 0 {
			return apperror.Unprocessable("Campaign tidak dapat dihapus: masih ada saldo donasi yang belum ditarik")
		}
	}

	pendingCount, err := s.donations.CountPendingByCampaign(ctx, id)
	if err != nil {
		return apperror.Internal("")
	}
	if pendingCount > 0 {
		return apperror.Unprocessable("Campaign tidak dapat dihapus: masih ada donasi yang sedang diproses")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return apperror.Internal("Gagal menghapus campaign")
	}
	s.audit.LogFinance(ctx, auditlog.Entry{
		Action: "campaign.deleted", Entity: "campaign", EntityID: id,
		Before: map[string]interface{}{"title": camp.Title},
	})
	return nil
}

func (s *ServiceImpl) CMSList(ctx context.Context, q dto.ListQuery, status string, categoryID int64) ([]campaign_dto.Response, int, error) {
	return s.list(ctx, campaign_dto.ListFilter{
		Status:     status,
		CategoryID: categoryID,
		Search:     q.Search,
		Limit:      q.Limit,
		Offset:     q.Offset(),
		OrderBy:    q.OrderBy(sortColumns, "c.createdDate DESC"),
	})
}

func (s *ServiceImpl) ListLite(ctx context.Context) ([]campaign_dto.LiteResponse, error) {
	rows, err := s.repo.ListLite(ctx)
	if err != nil {
		return nil, apperror.Internal("")
	}
	out := make([]campaign_dto.LiteResponse, 0, len(rows))
	for _, c := range rows {
		out = append(out, campaign_dto.LiteResponse{CampaignID: c.CampaignID, Title: c.Title})
	}
	return out, nil
}

func (s *ServiceImpl) Get(ctx context.Context, id int64) (campaign_dto.DetailResponse, error) {
	return s.getDetail(ctx, id)
}

func (s *ServiceImpl) Publish(ctx context.Context, id int64, caller CallerScope) (campaign_dto.DetailResponse, error) {
	camp, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return campaign_dto.DetailResponse{}, apperror.NotFound("Campaign tidak ditemukan")
	}
	return s.transition(ctx, camp, caller.UserID, constants.CampaignStatusPublished, "campaign.published", sql.NullString{},
		"Campaign hanya dapat dipublish saat berstatus draft")
}

func (s *ServiceImpl) Pause(ctx context.Context, id int64, caller CallerScope) (campaign_dto.DetailResponse, error) {
	camp, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return campaign_dto.DetailResponse{}, apperror.NotFound("Campaign tidak ditemukan")
	}
	return s.transition(ctx, camp, caller.UserID, constants.CampaignStatusPaused, "campaign.paused", sql.NullString{},
		"Campaign hanya dapat dijeda saat sedang tayang")
}

func (s *ServiceImpl) Resume(ctx context.Context, id int64, caller CallerScope) (campaign_dto.DetailResponse, error) {
	camp, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return campaign_dto.DetailResponse{}, apperror.NotFound("Campaign tidak ditemukan")
	}
	// Belum ada action spesifik "campaign.resumed" di techspec §16.1 (hanya
	// .published disebut) — dibedakan dari Publish() supaya audit trail
	// tidak menyamarkan resume sebagai publish pertama kali.
	return s.transition(ctx, camp, caller.UserID, constants.CampaignStatusPublished, "campaign.resumed", sql.NullString{},
		"Campaign hanya dapat dilanjutkan saat sedang dijeda")
}

func (s *ServiceImpl) Archive(ctx context.Context, id int64, caller CallerScope) (campaign_dto.DetailResponse, error) {
	camp, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return campaign_dto.DetailResponse{}, apperror.NotFound("Campaign tidak ditemukan")
	}
	return s.transition(ctx, camp, caller.UserID, constants.CampaignStatusArchived, "campaign.archived", sql.NullString{},
		"Campaign hanya dapat diarsipkan saat sudah selesai")
}

// ---------- helpers ----------

func (s *ServiceImpl) transition(ctx context.Context, camp campaign_model.Campaign, actorUserID int64, toStatus, action string, note sql.NullString, invalidMsg string) (campaign_dto.DetailResponse, error) {
	if !isValidTransition(camp.Status, toStatus) {
		return campaign_dto.DetailResponse{}, apperror.InvalidStatusTransition(invalidMsg)
	}
	if err := s.repo.UpdateStatus(ctx, camp.CampaignID, toStatus, note, actorUserID); err != nil {
		return campaign_dto.DetailResponse{}, apperror.Internal("")
	}
	s.audit.LogFinance(ctx, auditlog.Entry{
		ActorUserID: actorUserID, Action: action, Entity: "campaign", EntityID: camp.CampaignID,
		Before: map[string]string{"status": camp.Status}, After: map[string]string{"status": toStatus},
	})
	return s.getDetail(ctx, camp.CampaignID)
}

func (s *ServiceImpl) list(ctx context.Context, f campaign_dto.ListFilter) ([]campaign_dto.Response, int, error) {
	rows, total, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	return toResponses(rows), int(total), nil
}

func (s *ServiceImpl) getDetail(ctx context.Context, id int64) (campaign_dto.DetailResponse, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return campaign_dto.DetailResponse{}, apperror.NotFound("Campaign tidak ditemukan")
	}
	return s.toDetail(ctx, c)
}

func (s *ServiceImpl) toDetail(ctx context.Context, c campaign_model.Campaign) (campaign_dto.DetailResponse, error) {
	images, err := s.repo.ListImages(ctx, c.CampaignID)
	if err != nil {
		return campaign_dto.DetailResponse{}, apperror.Internal("")
	}
	urls := make([]string, 0, len(images))
	for _, img := range images {
		urls = append(urls, img.ImageUrl)
	}
	return campaign_dto.DetailResponse{Response: toResponse(c), SupportingImageUrls: urls}, nil
}

func (s *ServiceImpl) validateCategory(ctx context.Context, categoryID int64) error {
	exists, err := s.repo.CategoryExists(ctx, categoryID)
	if err != nil {
		return apperror.Internal("")
	}
	if !exists {
		return apperror.BadRequest("Kategori tidak ditemukan")
	}
	return nil
}

// resolveOrganization memvalidasi caller punya akses ke organisasi yang
// ditautkan ke campaign — mencegah campaign diklaim atas nama organisasi
// yang bukan wewenang caller.
func (s *ServiceImpl) resolveOrganization(ctx context.Context, caller CallerScope, organizationID *int64) (sql.NullInt64, error) {
	if organizationID == nil {
		return sql.NullInt64{}, nil
	}
	ok, err := s.orgAccess.IsAccessible(ctx, caller.OrganizationID, caller.OrganizationTypeCode, caller.WildcardTierAccess, *organizationID)
	if err != nil {
		return sql.NullInt64{}, apperror.Internal("")
	}
	if !ok {
		return sql.NullInt64{}, apperror.Forbidden("Anda tidak memiliki akses ke organisasi ini")
	}
	return sql.NullInt64{Int64: *organizationID, Valid: true}, nil
}

func (s *ServiceImpl) uniqueSlug(ctx context.Context, title string, exceptID int64) (string, error) {
	base := slug.Make(title)
	candidate := base
	for i := 2; i < 100; i++ {
		exists, err := s.repo.SlugExists(ctx, candidate, exceptID)
		if err != nil {
			return "", apperror.Internal("")
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
	return fmt.Sprintf("%s-%d", base, exceptID), nil
}

func toResponse(c campaign_model.Campaign) campaign_dto.Response {
	resp := campaign_dto.Response{
		CampaignID:         c.CampaignID,
		PublicRef:          c.PublicRef,
		Slug:                c.Slug,
		Title:               c.Title,
		CategoryID:          c.CategoryID,
		CategoryName:        c.CategoryName,
		Story:               c.Story,
		Goals:               c.Goals,
		CoverImageUrl:       c.CoverImageUrl,
		TargetAmount:        c.TargetAmount,
		CollectedAmount:     c.CollectedAmountCache,
		PicName:             c.PicName,
		PicPhone:            c.PicPhone,
		Status:              c.Status,
		IsFeatured:          c.IsFeatured,
		IsAnonymousAllowed:  c.IsAnonymousAllowed,
		HasDonations:        c.HasDonations,
		CreatedDate:         c.CreatedDate,
	}
	if c.OrganizationID.Valid {
		id := c.OrganizationID.Int64
		resp.OrganizationID = &id
	}
	if c.OrganizationName.Valid {
		name := c.OrganizationName.String
		resp.OrganizationName = &name
	}
	if c.ProvinceName.Valid {
		resp.ProvinceName = c.ProvinceName.String
	}
	if c.CityName.Valid {
		resp.CityName = c.CityName.String
	}
	if c.OrganizationNameOverride.Valid {
		resp.OrganizationNameOverride = c.OrganizationNameOverride.String
	}
	if c.OrganizationLogoUrl.Valid {
		resp.OrganizationLogoUrl = c.OrganizationLogoUrl.String
	}
	if c.OrganizationLinkUrl.Valid {
		resp.OrganizationLinkUrl = c.OrganizationLinkUrl.String
	}
	if c.LatestUpdate.Valid {
		resp.LatestUpdate = c.LatestUpdate.String
	}
	if c.StartDate.Valid {
		t := c.StartDate.Time
		resp.StartDate = &t
	}
	if c.EndDate.Valid {
		t := c.EndDate.Time
		resp.EndDate = &t
	}
	if c.ModerationNote.Valid {
		resp.ModerationNote = c.ModerationNote.String
	}
	return resp
}

func toResponses(rows []campaign_model.Campaign) []campaign_dto.Response {
	out := make([]campaign_dto.Response, 0, len(rows))
	for _, c := range rows {
		out = append(out, toResponse(c))
	}
	return out
}

func parseDateRange(startStr, endStr *string) (sql.NullTime, sql.NullTime, error) {
	start, err := parseDate(startStr)
	if err != nil {
		return sql.NullTime{}, sql.NullTime{}, apperror.BadRequest("Format startDate tidak valid (YYYY-MM-DD)")
	}
	end, err := parseDate(endStr)
	if err != nil {
		return sql.NullTime{}, sql.NullTime{}, apperror.BadRequest("Format endDate tidak valid (YYYY-MM-DD)")
	}
	if start.Valid && end.Valid && !end.Time.After(start.Time) {
		return sql.NullTime{}, sql.NullTime{}, apperror.BadRequest("endDate harus setelah startDate")
	}
	return start, end, nil
}

func parseDate(s *string) (sql.NullTime, error) {
	if s == nil || *s == "" {
		return sql.NullTime{}, nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return sql.NullTime{}, err
	}
	return sql.NullTime{Time: t, Valid: true}, nil
}

func nullStringFrom(s string) sql.NullString {
	s = strings.TrimSpace(s)
	return sql.NullString{String: s, Valid: s != ""}
}

func boolOrDefault(b *bool, def bool) bool {
	if b == nil {
		return def
	}
	return *b
}
