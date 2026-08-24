package wallet_service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"fsldk-api/base/apperror"
	"fsldk-api/constants"
	"fsldk-api/modules/campaign/campaign_dto"
	"fsldk-api/modules/campaign/campaign_model"
	"fsldk-api/modules/wallet/wallet_dto"
)

// fakeCampaignRepository adalah implementasi campaign_repository.Repository
// minimal untuk menguji validasi kepemilikan campaign di wallet_service.
type fakeCampaignRepository struct {
	byID map[int64]campaign_model.Campaign
}

func (f *fakeCampaignRepository) List(ctx context.Context, filter campaign_dto.ListFilter) ([]campaign_model.Campaign, int64, error) {
	return nil, 0, nil
}
func (f *fakeCampaignRepository) FindByID(ctx context.Context, id int64) (campaign_model.Campaign, error) {
	c, ok := f.byID[id]
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
	return nil
}
func (f *fakeCampaignRepository) UpdateStatus(ctx context.Context, id int64, status string, note sql.NullString, updatedBy int64) error {
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

func TestGetBalanceForOwner_RejectsNonOwnerWithNotFound(t *testing.T) {
	campRepo := &fakeCampaignRepository{byID: map[int64]campaign_model.Campaign{
		1: {CampaignID: 1, OwnerUserID: 10},
	}}
	svc := NewService(&fakeRepository{balance: 50000}, campRepo, nil)

	_, err := svc.GetBalanceForOwner(context.Background(), 1, 99)
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeNotFound {
		t.Fatalf("expected NotFound (IDOR-safe) for non-owner, got %v", err)
	}
}

func TestGetBalanceForOwner_AllowsActualOwner(t *testing.T) {
	campRepo := &fakeCampaignRepository{byID: map[int64]campaign_model.Campaign{
		1: {CampaignID: 1, OwnerUserID: 10},
	}}
	svc := NewService(&fakeRepository{balance: 50000}, campRepo, nil)

	resp, err := svc.GetBalanceForOwner(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("expected owner to access their own campaign balance, got error: %v", err)
	}
	if resp.AvailableBalance != 50000 {
		t.Fatalf("availableBalance = %v, want 50000", resp.AvailableBalance)
	}
}

func TestGetBalanceForOwner_UnknownCampaignReturnsNotFound(t *testing.T) {
	campRepo := &fakeCampaignRepository{byID: map[int64]campaign_model.Campaign{}}
	svc := NewService(&fakeRepository{}, campRepo, nil)

	_, err := svc.GetBalanceForOwner(context.Background(), 999, 10)
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeNotFound {
		t.Fatalf("expected NotFound for unknown campaign, got %v", err)
	}
}

func TestListLedgerForOwner_RejectsNonOwner(t *testing.T) {
	campRepo := &fakeCampaignRepository{byID: map[int64]campaign_model.Campaign{
		1: {CampaignID: 1, OwnerUserID: 10},
	}}
	svc := NewService(&fakeRepository{}, campRepo, nil)

	_, _, err := svc.ListLedgerForOwner(context.Background(), 1, 99, wallet_dto.LedgerListFilter{})
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeNotFound {
		t.Fatalf("expected NotFound for non-owner ledger access, got %v", err)
	}
}
