package campaign_service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"fsldk-api/base/apperror"
	"fsldk-api/constants"
	"fsldk-api/modules/campaign/campaign_dto"
	"fsldk-api/modules/campaign/campaign_model"
	"fsldk-api/modules/jobqueue/jobqueue_dto"
	"fsldk-api/modules/user/user_dto"
	"fsldk-api/modules/user/user_model"
	"fsldk-api/pkg/auditlog"
)

// fakeCampaignRepository adalah implementasi campaign_repository.Repository
// in-memory untuk menguji business logic campaign_service tanpa DB
// sungguhan — hanya method yang benar-benar dipakai skenario uji di file
// ini yang punya perilaku bermakna, sisanya no-op.
type fakeCampaignRepository struct {
	campaigns map[int64]campaign_model.Campaign
}

func newFakeRepo(c campaign_model.Campaign) *fakeCampaignRepository {
	return &fakeCampaignRepository{campaigns: map[int64]campaign_model.Campaign{c.CampaignID: c}}
}

func (f *fakeCampaignRepository) List(ctx context.Context, filter campaign_dto.ListFilter) ([]campaign_model.Campaign, int64, error) {
	return nil, 0, nil
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
func (f *fakeCampaignRepository) UpdateBeneficiary(ctx context.Context, id int64, p campaign_model.UpdateBeneficiaryParams) error {
	c := f.campaigns[id]
	c.BeneficiaryName = p.BeneficiaryName
	c.BeneficiaryBankCode = p.BeneficiaryBankCode
	c.BeneficiaryAccountNumber = p.BeneficiaryAccountNumber
	c.BeneficiaryAccountHolder = p.BeneficiaryAccountHolder
	c.BeneficiaryLockedUntil = sql.NullTime{Time: p.LockedUntil, Valid: true}
	f.campaigns[id] = c
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

func (f *fakeCampaignRepository) CreateReview(ctx context.Context, p campaign_model.ReviewParams) (int64, error) {
	return 0, nil
}

func (f *fakeCampaignRepository) ListReviews(ctx context.Context, campaignID int64) ([]campaign_model.Review, error) {
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

// fakeUserRepository adalah implementasi user_repository.Repository no-op —
// tidak ada skenario uji di file ini yang menegaskan isi notifikasi WA,
// jadi FindByID selalu "not found" cukup (notifyOwner no-op diam-diam).
type fakeUserRepository struct{}

func (f *fakeUserRepository) FindByID(ctx context.Context, id int64) (user_model.User, error) {
	return user_model.User{}, errors.New("not found")
}
func (f *fakeUserRepository) FindByEmail(ctx context.Context, email string) (user_model.User, error) {
	return user_model.User{}, errors.New("not found")
}
func (f *fakeUserRepository) FindByGoogleID(ctx context.Context, googleID string) (user_model.User, error) {
	return user_model.User{}, errors.New("not found")
}
func (f *fakeUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return false, nil
}
func (f *fakeUserRepository) ExistsByEmailExcept(ctx context.Context, email string, exceptID int64) (bool, error) {
	return false, nil
}
func (f *fakeUserRepository) Create(ctx context.Context, p user_model.CreateParams) (int64, error) {
	return 0, nil
}
func (f *fakeUserRepository) List(ctx context.Context, filt user_dto.ListFilter) ([]user_model.User, int64, error) {
	return nil, 0, nil
}
func (f *fakeUserRepository) SearchActive(ctx context.Context, search string, limit int) ([]user_model.User, error) {
	return nil, nil
}
func (f *fakeUserRepository) Update(ctx context.Context, id int64, fullName, email string, roleID int64, isActive bool, organizationID sql.NullInt64, wildcardTierAccess sql.NullString, updatedBy int64) error {
	return nil
}
func (f *fakeUserRepository) UpdateContactInfo(ctx context.Context, id int64, phoneNumber, address sql.NullString) error {
	return nil
}
func (f *fakeUserRepository) SetActive(ctx context.Context, id int64, active bool, updatedBy int64) error {
	return nil
}
func (f *fakeUserRepository) SetPassword(ctx context.Context, id int64, hashed string, mustChange bool) error {
	return nil
}
func (f *fakeUserRepository) LinkGoogle(ctx context.Context, id int64, googleID string, markVerified bool) error {
	return nil
}
func (f *fakeUserRepository) UpdatePhoto(ctx context.Context, id int64, photoURL string) error {
	return nil
}
func (f *fakeUserRepository) UpdateCustomPhoto(ctx context.Context, id int64, photoURL string) error {
	return nil
}
func (f *fakeUserRepository) MarkEmailVerified(ctx context.Context, id int64) error { return nil }
func (f *fakeUserRepository) SoftDelete(ctx context.Context, id int64, updatedBy int64) error {
	return nil
}
func (f *fakeUserRepository) LogLogin(ctx context.Context, userID int64, ip, ua, status string) error {
	return nil
}

// fakeJobEnqueuer adalah implementasi JobEnqueuer no-op — notifikasi WA
// tidak relevan diverifikasi di unit test business-logic ini.
type fakeJobEnqueuer struct{}

func (f *fakeJobEnqueuer) Enqueue(ctx context.Context, in jobqueue_dto.EnqueueInput) (int64, error) {
	return 0, nil
}

const validStory = "cerita panjang minimal lima puluh karakter untuk lolos validasi story campaign ini"

func TestUpdate_RejectsNonOwnerAsNotFound(t *testing.T) {
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, OwnerUserID: 10, Status: constants.CampaignStatusDraft})
	svc := NewService(repo, &fakeUserRepository{}, fakeOrgAccess{allow: true}, &fakeJobEnqueuer{}, &fakeFinanceAuditor{})

	_, err := svc.Update(context.Background(), 1, CallerScope{UserID: 99}, campaign_dto.UpdateRequest{Title: "Judul Campaign Baru", Story: validStory})
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeNotFound {
		t.Fatalf("expected NotFound error for non-owner update (IDOR-safe), got %v", err)
	}
}

func TestUpdate_RejectsWhenStatusNotEditable(t *testing.T) {
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, OwnerUserID: 10, Status: constants.CampaignStatusPublished})
	svc := NewService(repo, &fakeUserRepository{}, fakeOrgAccess{allow: true}, &fakeJobEnqueuer{}, &fakeFinanceAuditor{})

	_, err := svc.Update(context.Background(), 1, CallerScope{UserID: 10}, campaign_dto.UpdateRequest{Title: "Judul Campaign Baru", Story: validStory})
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeInvalidStatusTransition {
		t.Fatalf("expected InvalidStatusTransition error when editing a published campaign, got %v", err)
	}
}

func TestUpdate_AllowsOwnerDuringRevisionRequested(t *testing.T) {
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, OwnerUserID: 10, Status: constants.CampaignStatusRevisionRequested})
	svc := NewService(repo, &fakeUserRepository{}, fakeOrgAccess{allow: true}, &fakeJobEnqueuer{}, &fakeFinanceAuditor{})

	_, err := svc.Update(context.Background(), 1, CallerScope{UserID: 10}, campaign_dto.UpdateRequest{Title: "Judul Campaign Baru", Story: validStory})
	if err != nil {
		t.Fatalf("expected owner to edit a revision-requested campaign, got error: %v", err)
	}
}

func TestSubmit_RejectsNonOwnerAsNotFound(t *testing.T) {
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, OwnerUserID: 10, Status: constants.CampaignStatusDraft})
	svc := NewService(repo, &fakeUserRepository{}, fakeOrgAccess{allow: true}, &fakeJobEnqueuer{}, &fakeFinanceAuditor{})

	_, err := svc.Submit(context.Background(), 1, CallerScope{UserID: 99})
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeNotFound {
		t.Fatalf("expected NotFound error for non-owner submit (IDOR-safe), got %v", err)
	}
}

func TestSubmit_SucceedsFromDraft(t *testing.T) {
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, OwnerUserID: 10, Status: constants.CampaignStatusDraft})
	svc := NewService(repo, &fakeUserRepository{}, fakeOrgAccess{allow: true}, &fakeJobEnqueuer{}, &fakeFinanceAuditor{})

	resp, err := svc.Submit(context.Background(), 1, CallerScope{UserID: 10})
	if err != nil {
		t.Fatalf("expected submit from DRAFT to succeed, got error: %v", err)
	}
	if resp.Status != constants.CampaignStatusSubmitted {
		t.Fatalf("expected status SUBMITTED after submit, got %s", resp.Status)
	}
}

func TestSubmit_RejectsFromPublished(t *testing.T) {
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, OwnerUserID: 10, Status: constants.CampaignStatusPublished})
	svc := NewService(repo, &fakeUserRepository{}, fakeOrgAccess{allow: true}, &fakeJobEnqueuer{}, &fakeFinanceAuditor{})

	_, err := svc.Submit(context.Background(), 1, CallerScope{UserID: 10})
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeInvalidStatusTransition {
		t.Fatalf("expected InvalidStatusTransition error submitting an already-published campaign, got %v", err)
	}
}

func TestReview_RejectsInvalidDecision(t *testing.T) {
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, Status: constants.CampaignStatusSubmitted})
	svc := NewService(repo, &fakeUserRepository{}, fakeOrgAccess{allow: true}, &fakeJobEnqueuer{}, &fakeFinanceAuditor{})

	_, err := svc.Review(context.Background(), 1, CallerScope{UserID: 5}, campaign_dto.ReviewRequest{Decision: "MAYBE"})
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeValidationError {
		t.Fatalf("expected validation error for unrecognized decision, got %v", err)
	}
}

func TestReview_RejectsWhenNotInReviewableStatus(t *testing.T) {
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, Status: constants.CampaignStatusDraft})
	svc := NewService(repo, &fakeUserRepository{}, fakeOrgAccess{allow: true}, &fakeJobEnqueuer{}, &fakeFinanceAuditor{})

	_, err := svc.Review(context.Background(), 1, CallerScope{UserID: 5}, campaign_dto.ReviewRequest{Decision: constants.ReviewDecisionApproved})
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeInvalidStatusTransition {
		t.Fatalf("expected InvalidStatusTransition error reviewing a draft campaign, got %v", err)
	}
}

func TestReview_ApprovedMovesToApprovedStatus(t *testing.T) {
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, Status: constants.CampaignStatusSubmitted})
	svc := NewService(repo, &fakeUserRepository{}, fakeOrgAccess{allow: true}, &fakeJobEnqueuer{}, &fakeFinanceAuditor{})

	resp, err := svc.Review(context.Background(), 1, CallerScope{UserID: 5}, campaign_dto.ReviewRequest{Decision: constants.ReviewDecisionApproved})
	if err != nil {
		t.Fatalf("expected review to succeed, got error: %v", err)
	}
	if resp.Status != constants.CampaignStatusApproved {
		t.Fatalf("expected status APPROVED after approving review, got %s", resp.Status)
	}
}

func TestCreate_RejectsUnknownCategory(t *testing.T) {
	repo := &fakeCampaignRepository{campaigns: map[int64]campaign_model.Campaign{}}
	catRejecting := &categoryRejectingRepo{fakeCampaignRepository: repo}
	svc := NewService(catRejecting, &fakeUserRepository{}, fakeOrgAccess{allow: true}, &fakeJobEnqueuer{}, &fakeFinanceAuditor{})

	_, err := svc.Create(context.Background(), CallerScope{UserID: 1}, campaign_dto.CreateRequest{
		Title: "Judul Campaign Baru", CategoryID: 999, Story: validStory, CoverImageUrl: "/uploads/cover.jpg",
		TargetAmount: 100000, BeneficiaryName: "A", BeneficiaryBankCode: "bca", BeneficiaryAccountNumber: "123", BeneficiaryAccountHolder: "A",
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

func TestUpdateBeneficiary_RejectsNonOwnerAsNotFound(t *testing.T) {
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, OwnerUserID: 10, Status: constants.CampaignStatusPublished})
	svc := NewService(repo, &fakeUserRepository{}, fakeOrgAccess{allow: true}, &fakeJobEnqueuer{}, &fakeFinanceAuditor{})

	_, err := svc.UpdateBeneficiary(context.Background(), 1, CallerScope{UserID: 99}, campaign_dto.UpdateBeneficiaryRequest{
		BeneficiaryName: "A", BeneficiaryBankCode: "bca", BeneficiaryAccountNumber: "999", BeneficiaryAccountHolder: "A",
	})
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeNotFound {
		t.Fatalf("expected NotFound (IDOR-safe) for non-owner beneficiary change, got %v", err)
	}
}

func TestUpdateBeneficiary_OwnerSucceedsAndLocksForCoolingPeriod(t *testing.T) {
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, OwnerUserID: 10, Status: constants.CampaignStatusPublished})
	svc := NewService(repo, &fakeUserRepository{}, fakeOrgAccess{allow: true}, &fakeJobEnqueuer{}, &fakeFinanceAuditor{})

	before := time.Now()
	_, err := svc.UpdateBeneficiary(context.Background(), 1, CallerScope{UserID: 10}, campaign_dto.UpdateBeneficiaryRequest{
		BeneficiaryName: "Rekening Baru", BeneficiaryBankCode: "bni", BeneficiaryAccountNumber: "999888777", BeneficiaryAccountHolder: "Rekening Baru",
	})
	if err != nil {
		t.Fatalf("expected owner to change beneficiary successfully, got error: %v", err)
	}
	updated := repo.campaigns[1]
	if updated.BeneficiaryBankCode != "bni" || updated.BeneficiaryAccountNumber != "999888777" {
		t.Fatalf("beneficiary fields not updated: %+v", updated)
	}
	if !updated.BeneficiaryLockedUntil.Valid || !updated.BeneficiaryLockedUntil.Time.After(before.Add(23*time.Hour)) {
		t.Fatalf("expected beneficiaryLockedUntil to be set ~24h in the future, got %+v", updated.BeneficiaryLockedUntil)
	}
}
