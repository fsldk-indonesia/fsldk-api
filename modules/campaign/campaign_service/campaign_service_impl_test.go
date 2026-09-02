package campaign_service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"gorm.io/gorm"

	"fsldk-api/base/apperror"
	"fsldk-api/constants"
	"fsldk-api/modules/campaign/campaign_dto"
	"fsldk-api/modules/campaign/campaign_model"
	"fsldk-api/modules/wallet/wallet_dto"
	"fsldk-api/pkg/auditlog"
)

// fakeCampaignRepository adalah implementasi campaign_repository.Repository
// in-memory untuk menguji business logic campaign_service tanpa DB
// sungguhan — hanya method yang benar-benar dipakai skenario uji di file
// ini yang punya perilaku bermakna, sisanya no-op.
type fakeCampaignRepository struct {
	campaigns map[int64]campaign_model.Campaign
	deleted   []int64
}

func newFakeRepo(c campaign_model.Campaign) *fakeCampaignRepository {
	return &fakeCampaignRepository{campaigns: map[int64]campaign_model.Campaign{c.CampaignID: c}}
}

func (f *fakeCampaignRepository) List(ctx context.Context, filter campaign_dto.ListFilter) ([]campaign_model.Campaign, int64, error) {
	return nil, 0, nil
}

func (f *fakeCampaignRepository) ListLite(ctx context.Context) ([]campaign_model.Campaign, error) {
	return nil, nil
}

func (f *fakeCampaignRepository) FindByID(ctx context.Context, id int64) (campaign_model.Campaign, error) {
	c, ok := f.campaigns[id]
	if !ok {
		return campaign_model.Campaign{}, errors.New("not found")
	}
	return c, nil
}

func (f *fakeCampaignRepository) FindBySlug(ctx context.Context, slug string) (campaign_model.Campaign, error) {
	return campaign_model.Campaign{}, errors.New("not found")
}

func (f *fakeCampaignRepository) SlugExists(ctx context.Context, slug string, exceptID int64) (bool, error) {
	return false, nil
}

func (f *fakeCampaignRepository) CategoryExists(ctx context.Context, categoryID int64) (bool, error) {
	return true, nil
}

func (f *fakeCampaignRepository) Categories(ctx context.Context) ([]campaign_model.Category, error) {
	return nil, nil
}

func (f *fakeCampaignRepository) Create(ctx context.Context, p campaign_model.CreateParams) (int64, error) {
	return 0, nil
}

func (f *fakeCampaignRepository) Update(ctx context.Context, id int64, p campaign_model.UpdateParams) error {
	return nil
}

func (f *fakeCampaignRepository) Delete(ctx context.Context, id int64) error {
	f.deleted = append(f.deleted, id)
	delete(f.campaigns, id)
	return nil
}

func (f *fakeCampaignRepository) UpdateStatus(ctx context.Context, id int64, status string, note sql.NullString, updatedBy int64) error {
	c := f.campaigns[id]
	c.Status = status
	f.campaigns[id] = c
	return nil
}

func (f *fakeCampaignRepository) ReplaceImages(ctx context.Context, campaignID int64, urls []string) error {
	return nil
}

func (f *fakeCampaignRepository) ListImages(ctx context.Context, campaignID int64) ([]campaign_model.Image, error) {
	return nil, nil
}

type fakeOrgAccess struct{ allow bool }

func (f fakeOrgAccess) IsAccessible(ctx context.Context, callerOrganizationID *int64, callerOrganizationTypeCode, wildcardTierAccess string, targetOrganizationID int64) (bool, error) {
	return f.allow, nil
}

// fakeFinanceAuditor adalah implementasi FinanceAuditor no-op — isi audit
// trail tidak relevan diverifikasi di unit test business-logic ini.
type fakeFinanceAuditor struct{}

func (f *fakeFinanceAuditor) LogFinance(ctx context.Context, e auditlog.Entry) {}

// fakeDonationChecker/fakeWithdrawalChecker adalah implementasi no-op untuk
// guard Delete() — default "tidak ada donasi/withdrawal aktif" (boleh
// dihapus), override field untuk skenario uji Delete() spesifik.
type fakeDonationChecker struct {
	paidCount, pendingCount int64
}

func (f *fakeDonationChecker) CountPaidByCampaign(ctx context.Context, campaignID int64) (int64, error) {
	return f.paidCount, nil
}
func (f *fakeDonationChecker) CountPendingByCampaign(ctx context.Context, campaignID int64) (int64, error) {
	return f.pendingCount, nil
}

type fakeWithdrawalChecker struct{ nonFinalCount int64 }

func (f *fakeWithdrawalChecker) CountNonFinalByCampaign(ctx context.Context, campaignID int64) (int64, error) {
	return f.nonFinalCount, nil
}

// fakeWalletService adalah implementasi wallet_service.Service no-op — hanya
// GetBalance yang punya perilaku bermakna (dipakai guard Delete()), method
// lain tidak pernah dipanggil skenario uji campaign_service.
type fakeWalletService struct{ availableBalance float64 }

func (f *fakeWalletService) CreditDonation(tx *gorm.DB, campaignID, donationID int64, amount float64, note string) error {
	return nil
}
func (f *fakeWalletService) ReserveWithdrawal(tx *gorm.DB, campaignID, withdrawalID int64, amount float64, actorUserID int64, note string) error {
	return nil
}
func (f *fakeWalletService) ReleaseWithdrawal(tx *gorm.DB, campaignID, withdrawalID int64, amount float64, note string) error {
	return nil
}
func (f *fakeWalletService) RefundDebit(tx *gorm.DB, campaignID, donationID int64, amount float64, actorUserID int64, note string) error {
	return nil
}
func (f *fakeWalletService) AdjustBalance(ctx context.Context, campaignID int64, amount float64, direction string, actorUserID int64, reason string) error {
	return nil
}
func (f *fakeWalletService) GetBalance(ctx context.Context, campaignID int64) (wallet_dto.BalanceResponse, error) {
	return wallet_dto.BalanceResponse{AvailableBalance: f.availableBalance}, nil
}
func (f *fakeWalletService) ListLedger(ctx context.Context, campaignID int64, filter wallet_dto.LedgerListFilter) ([]wallet_dto.LedgerListItem, int64, error) {
	return nil, 0, nil
}

// newTestService membangun ServiceImpl dengan seluruh dependency no-op —
// dipakai default oleh skenario uji yang tidak menegaskan guard Delete().
func newTestService(repo *fakeCampaignRepository) Service {
	return NewService(repo, fakeOrgAccess{allow: true}, &fakeFinanceAuditor{}, &fakeDonationChecker{}, &fakeWithdrawalChecker{}, &fakeWalletService{})
}

const validStory = "cerita panjang minimal lima puluh karakter untuk lolos validasi story campaign ini"
const validGoals = "tujuan penggunaan dana yang jelas"

func TestUpdate_RejectsWhenArchived(t *testing.T) {
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, Status: constants.CampaignStatusArchived})
	svc := newTestService(repo)

	_, err := svc.Update(context.Background(), 1, CallerScope{UserID: 10}, campaign_dto.UpdateRequest{Title: "Judul Campaign Baru", Story: validStory, Goals: validGoals, PicName: "PIC", PicPhone: "6281234567890"})
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeInvalidStatusTransition {
		t.Fatalf("expected InvalidStatusTransition error when editing an archived campaign, got %v", err)
	}
}

func TestUpdate_AllowsAnyPermittedCallerWhenPublished(t *testing.T) {
	// Campaign murni CRUD berbasis permission (revisi 2026-09-01) — tidak
	// ada lagi kepemilikan, siapapun dengan akses boleh mengubah campaign
	// manapun kecuali ARCHIVED.
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, Status: constants.CampaignStatusPublished})
	svc := newTestService(repo)

	_, err := svc.Update(context.Background(), 1, CallerScope{UserID: 999}, campaign_dto.UpdateRequest{Title: "Judul Campaign Baru", Story: validStory, Goals: validGoals, PicName: "PIC", PicPhone: "6281234567890"})
	if err != nil {
		t.Fatalf("expected any permitted caller to edit a published campaign, got error: %v", err)
	}
}

func TestPublish_SucceedsFromDraft(t *testing.T) {
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, Status: constants.CampaignStatusDraft})
	svc := newTestService(repo)

	resp, err := svc.Publish(context.Background(), 1, CallerScope{UserID: 5})
	if err != nil {
		t.Fatalf("expected publish from DRAFT to succeed, got error: %v", err)
	}
	if resp.Status != constants.CampaignStatusPublished {
		t.Fatalf("expected status PUBLISHED after publish, got %s", resp.Status)
	}
}

func TestPublish_RejectsWhenAlreadyPublished(t *testing.T) {
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, Status: constants.CampaignStatusPublished})
	svc := newTestService(repo)

	_, err := svc.Publish(context.Background(), 1, CallerScope{UserID: 5})
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeInvalidStatusTransition {
		t.Fatalf("expected InvalidStatusTransition error re-publishing an already-published campaign, got %v", err)
	}
}

func TestArchive_SucceedsFromPublished(t *testing.T) {
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, Status: constants.CampaignStatusPublished})
	svc := newTestService(repo)

	resp, err := svc.Archive(context.Background(), 1, CallerScope{UserID: 5})
	if err != nil {
		t.Fatalf("expected archive from PUBLISHED to succeed, got error: %v", err)
	}
	if resp.Status != constants.CampaignStatusArchived {
		t.Fatalf("expected status ARCHIVED after archive, got %s", resp.Status)
	}
}

func TestCreate_RejectsUnknownCategory(t *testing.T) {
	repo := &fakeCampaignRepository{campaigns: map[int64]campaign_model.Campaign{}}
	catRejecting := &categoryRejectingRepo{fakeCampaignRepository: repo}
	svc := NewService(catRejecting, fakeOrgAccess{allow: true}, &fakeFinanceAuditor{}, &fakeDonationChecker{}, &fakeWithdrawalChecker{}, &fakeWalletService{})

	_, err := svc.Create(context.Background(), CallerScope{UserID: 1}, campaign_dto.CreateRequest{
		Title: "Judul Campaign Baru", CategoryID: 999, Story: validStory, Goals: validGoals, CoverImageUrl: "/uploads/cover.jpg",
		TargetAmount: 100000, PicName: "PIC", PicPhone: "6281234567890",
	})
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeValidationError {
		t.Fatalf("expected validation error for unknown category, got %v", err)
	}
}

// categoryRejectingRepo membungkus fakeCampaignRepository dan selalu
// melaporkan kategori tidak ditemukan — dipakai satu skenario uji di atas.
type categoryRejectingRepo struct {
	*fakeCampaignRepository
}

func (r *categoryRejectingRepo) CategoryExists(ctx context.Context, categoryID int64) (bool, error) {
	return false, nil
}

func TestDelete_SucceedsWhenNoDonations(t *testing.T) {
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, Status: constants.CampaignStatusDraft})
	svc := newTestService(repo)

	if err := svc.Delete(context.Background(), 1); err != nil {
		t.Fatalf("expected delete to succeed for campaign with no donations, got error: %v", err)
	}
	if _, ok := repo.campaigns[1]; ok {
		t.Fatalf("expected campaign to be removed from repository")
	}
}

func TestDelete_RejectsWhenPendingDonationsExist(t *testing.T) {
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, Status: constants.CampaignStatusPublished})
	svc := NewService(repo, fakeOrgAccess{allow: true}, &fakeFinanceAuditor{}, &fakeDonationChecker{pendingCount: 1}, &fakeWithdrawalChecker{}, &fakeWalletService{})

	err := svc.Delete(context.Background(), 1)
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeUnprocessable {
		t.Fatalf("expected Unprocessable error when pending donations exist, got %v", err)
	}
}

func TestDelete_RejectsWhenNonFinalWithdrawalExists(t *testing.T) {
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, Status: constants.CampaignStatusPublished})
	svc := NewService(repo, fakeOrgAccess{allow: true}, &fakeFinanceAuditor{}, &fakeDonationChecker{paidCount: 1}, &fakeWithdrawalChecker{nonFinalCount: 1}, &fakeWalletService{})

	err := svc.Delete(context.Background(), 1)
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeUnprocessable {
		t.Fatalf("expected Unprocessable error when a non-final withdrawal exists, got %v", err)
	}
}

func TestDelete_RejectsWhenBalanceUnwithdrawn(t *testing.T) {
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, Status: constants.CampaignStatusPublished})
	svc := NewService(repo, fakeOrgAccess{allow: true}, &fakeFinanceAuditor{}, &fakeDonationChecker{paidCount: 1}, &fakeWithdrawalChecker{}, &fakeWalletService{availableBalance: 50000})

	err := svc.Delete(context.Background(), 1)
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeUnprocessable {
		t.Fatalf("expected Unprocessable error when balance is not fully withdrawn, got %v", err)
	}
}

func TestDelete_SucceedsWhenPaidButFullyWithdrawn(t *testing.T) {
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, Status: constants.CampaignStatusPublished})
	svc := NewService(repo, fakeOrgAccess{allow: true}, &fakeFinanceAuditor{}, &fakeDonationChecker{paidCount: 1}, &fakeWithdrawalChecker{}, &fakeWalletService{availableBalance: 0})

	if err := svc.Delete(context.Background(), 1); err != nil {
		t.Fatalf("expected delete to succeed once balance is fully withdrawn, got error: %v", err)
	}
}
