package campaign_service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"fsldk-api/base/apperror"
	"fsldk-api/constants"
	"fsldk-api/modules/campaign/campaign_dto"
	"fsldk-api/modules/campaign/campaign_model"
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

const validStory = "cerita panjang minimal lima puluh karakter untuk lolos validasi story campaign ini"

func TestUpdate_RejectsNonOwnerAsNotFound(t *testing.T) {
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, OwnerUserID: 10, Status: constants.CampaignStatusDraft})
	svc := NewService(repo, fakeOrgAccess{allow: true})

	_, err := svc.Update(context.Background(), 1, CallerScope{UserID: 99}, campaign_dto.UpdateRequest{Title: "Judul Campaign Baru", Story: validStory})
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeNotFound {
		t.Fatalf("expected NotFound error for non-owner update (IDOR-safe), got %v", err)
	}
}

func TestUpdate_RejectsWhenStatusNotEditable(t *testing.T) {
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, OwnerUserID: 10, Status: constants.CampaignStatusPublished})
	svc := NewService(repo, fakeOrgAccess{allow: true})

	_, err := svc.Update(context.Background(), 1, CallerScope{UserID: 10}, campaign_dto.UpdateRequest{Title: "Judul Campaign Baru", Story: validStory})
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeInvalidStatusTransition {
		t.Fatalf("expected InvalidStatusTransition error when editing a published campaign, got %v", err)
	}
}

func TestUpdate_AllowsOwnerDuringRevisionRequested(t *testing.T) {
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, OwnerUserID: 10, Status: constants.CampaignStatusRevisionRequested})
	svc := NewService(repo, fakeOrgAccess{allow: true})

	_, err := svc.Update(context.Background(), 1, CallerScope{UserID: 10}, campaign_dto.UpdateRequest{Title: "Judul Campaign Baru", Story: validStory})
	if err != nil {
		t.Fatalf("expected owner to edit a revision-requested campaign, got error: %v", err)
	}
}

func TestSubmit_RejectsNonOwnerAsNotFound(t *testing.T) {
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, OwnerUserID: 10, Status: constants.CampaignStatusDraft})
	svc := NewService(repo, fakeOrgAccess{allow: true})

	_, err := svc.Submit(context.Background(), 1, CallerScope{UserID: 99})
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeNotFound {
		t.Fatalf("expected NotFound error for non-owner submit (IDOR-safe), got %v", err)
	}
}

func TestSubmit_SucceedsFromDraft(t *testing.T) {
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, OwnerUserID: 10, Status: constants.CampaignStatusDraft})
	svc := NewService(repo, fakeOrgAccess{allow: true})

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
	svc := NewService(repo, fakeOrgAccess{allow: true})

	_, err := svc.Submit(context.Background(), 1, CallerScope{UserID: 10})
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeInvalidStatusTransition {
		t.Fatalf("expected InvalidStatusTransition error submitting an already-published campaign, got %v", err)
	}
}

func TestReview_RejectsInvalidDecision(t *testing.T) {
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, Status: constants.CampaignStatusSubmitted})
	svc := NewService(repo, fakeOrgAccess{allow: true})

	_, err := svc.Review(context.Background(), 1, CallerScope{UserID: 5}, campaign_dto.ReviewRequest{Decision: "MAYBE"})
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeValidationError {
		t.Fatalf("expected validation error for unrecognized decision, got %v", err)
	}
}

func TestReview_RejectsWhenNotInReviewableStatus(t *testing.T) {
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, Status: constants.CampaignStatusDraft})
	svc := NewService(repo, fakeOrgAccess{allow: true})

	_, err := svc.Review(context.Background(), 1, CallerScope{UserID: 5}, campaign_dto.ReviewRequest{Decision: constants.ReviewDecisionApproved})
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeInvalidStatusTransition {
		t.Fatalf("expected InvalidStatusTransition error reviewing a draft campaign, got %v", err)
	}
}

func TestReview_ApprovedMovesToApprovedStatus(t *testing.T) {
	repo := newFakeRepo(campaign_model.Campaign{CampaignID: 1, Status: constants.CampaignStatusSubmitted})
	svc := NewService(repo, fakeOrgAccess{allow: true})

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
	svc := NewService(catRejecting, fakeOrgAccess{allow: true})

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
