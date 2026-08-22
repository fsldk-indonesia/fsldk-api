package donation_service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"fsldk-api/base/apperror"
	"fsldk-api/constants"
	"fsldk-api/modules/campaign/campaign_dto"
	"fsldk-api/modules/campaign/campaign_model"
	"fsldk-api/modules/donation/donation_dto"
	"fsldk-api/modules/donation/donation_model"
	"fsldk-api/modules/donation/donation_repository"
)

// fakeCampaignRepository adalah implementasi campaign_repository.Repository
// minimal untuk menguji donation_service — hanya FindBySlug yang berperilaku
// bermakna, sisanya no-op karena tidak dipakai skenario uji di file ini.
type fakeCampaignRepository struct {
	bySlug map[string]campaign_model.Campaign
}

func (f *fakeCampaignRepository) List(ctx context.Context, filter campaign_dto.ListFilter) ([]campaign_model.Campaign, int64, error) {
	return nil, 0, nil
}
func (f *fakeCampaignRepository) FindByID(ctx context.Context, id int64) (campaign_model.Campaign, error) {
	return campaign_model.Campaign{}, errors.New("not found")
}
func (f *fakeCampaignRepository) FindBySlug(ctx context.Context, slug string) (campaign_model.Campaign, error) {
	c, ok := f.bySlug[slug]
	if !ok {
		return campaign_model.Campaign{}, errors.New("not found")
	}
	return c, nil
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

// fakeDonationRepository adalah implementasi donation_repository.Repository
// in-memory — Create mendeteksi idempotencyKey duplikat sendiri (meniru
// guard UNIQUE di DB) supaya alur idempotent Create() bisa diuji tanpa DB.
type fakeDonationRepository struct {
	nextID       int64
	byID         map[int64]donation_model.Donation
	byIdempotent map[string]int64
	createCalls  int
}

func newFakeDonationRepo() *fakeDonationRepository {
	return &fakeDonationRepository{byID: map[int64]donation_model.Donation{}, byIdempotent: map[string]int64{}}
}

func (f *fakeDonationRepository) Create(ctx context.Context, p donation_model.CreateParams) (int64, error) {
	f.createCalls++
	if _, exists := f.byIdempotent[p.IdempotencyKey]; exists {
		return 0, donation_repository.ErrDuplicateIdempotencyKey
	}
	f.nextID++
	id := f.nextID
	f.byID[id] = donation_model.Donation{
		DonationID:     id,
		PublicRef:      p.PublicRef,
		CampaignID:     p.CampaignID,
		DonorName:      p.DonorName,
		IsAnonymous:    p.IsAnonymous,
		Amount:         p.Amount,
		AdminFee:       p.AdminFee,
		TotalAmount:    p.TotalAmount,
		PaymentStatus:  constants.DonationStatusPending,
		Gateway:        p.Gateway,
		IdempotencyKey: p.IdempotencyKey,
	}
	f.byIdempotent[p.IdempotencyKey] = id
	return id, nil
}

func (f *fakeDonationRepository) FindByID(ctx context.Context, id int64) (donation_model.Donation, error) {
	d, ok := f.byID[id]
	if !ok {
		return donation_model.Donation{}, donation_repository.ErrNotFound
	}
	return d, nil
}

func (f *fakeDonationRepository) FindByPublicRef(ctx context.Context, publicRef string) (donation_model.Donation, error) {
	for _, d := range f.byID {
		if d.PublicRef == publicRef {
			return d, nil
		}
	}
	return donation_model.Donation{}, donation_repository.ErrNotFound
}

func (f *fakeDonationRepository) FindByIdempotencyKey(ctx context.Context, key string) (donation_model.Donation, error) {
	id, ok := f.byIdempotent[key]
	if !ok {
		return donation_model.Donation{}, donation_repository.ErrNotFound
	}
	return f.byID[id], nil
}

func (f *fakeDonationRepository) List(ctx context.Context, filter donation_dto.ListFilter) ([]donation_model.Donation, int64, error) {
	return nil, 0, nil
}

func publishedCampaign(id int64, slug string) campaign_model.Campaign {
	return campaign_model.Campaign{
		CampaignID:         id,
		Slug:               slug,
		Status:             constants.CampaignStatusPublished,
		IsAnonymousAllowed: true,
	}
}

func TestCreate_RejectsWhenCampaignNotPublished(t *testing.T) {
	campRepo := &fakeCampaignRepository{bySlug: map[string]campaign_model.Campaign{
		"draft-campaign": {CampaignID: 1, Slug: "draft-campaign", Status: constants.CampaignStatusDraft},
	}}
	svc := NewService(newFakeDonationRepo(), campRepo, 1, 24)

	_, err := svc.Create(context.Background(), "draft-campaign", nil, donation_dto.CreateRequest{
		Amount: 20000, DonorName: "Budi", DonorEmail: "budi@example.com", DonorPhone: "0812",
	})
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeNotFound {
		t.Fatalf("expected NotFound error donating to a non-published campaign, got %v", err)
	}
}

func TestCreate_RejectsAnonymousWhenCampaignDisallows(t *testing.T) {
	camp := publishedCampaign(1, "c1")
	camp.IsAnonymousAllowed = false
	campRepo := &fakeCampaignRepository{bySlug: map[string]campaign_model.Campaign{"c1": camp}}
	svc := NewService(newFakeDonationRepo(), campRepo, 1, 24)

	_, err := svc.Create(context.Background(), "c1", nil, donation_dto.CreateRequest{
		Amount: 20000, DonorName: "Budi", DonorEmail: "budi@example.com", DonorPhone: "0812", IsAnonymous: true,
	})
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeValidationError {
		t.Fatalf("expected validation error for disallowed anonymous donation, got %v", err)
	}
}

func TestCreate_ComputesGoldenFeeFormula(t *testing.T) {
	campRepo := &fakeCampaignRepository{bySlug: map[string]campaign_model.Campaign{"c1": publishedCampaign(1, "c1")}}
	svc := NewService(newFakeDonationRepo(), campRepo, 1, 24)

	resp, err := svc.Create(context.Background(), "c1", nil, donation_dto.CreateRequest{
		Amount: 20000, DonorName: "Budi", DonorEmail: "budi@example.com", DonorPhone: "0812",
	})
	if err != nil {
		t.Fatalf("expected donation to be created, got error: %v", err)
	}
	if resp.TotalAmount != 20203 {
		t.Errorf("totalAmount = %v, want 20203 (golden MDR figure)", resp.TotalAmount)
	}
	if resp.AdminFee != 203 {
		t.Errorf("adminFee = %v, want 203 (golden MDR figure)", resp.AdminFee)
	}
	if resp.Amount != 20000 {
		t.Errorf("amount = %v, want 20000", resp.Amount)
	}
	if resp.PaymentStatus != constants.DonationStatusPending {
		t.Errorf("paymentStatus = %v, want PENDING", resp.PaymentStatus)
	}
}

func TestCreate_DuplicateIdempotencyKeyReturnsExistingDonation(t *testing.T) {
	campRepo := &fakeCampaignRepository{bySlug: map[string]campaign_model.Campaign{"c1": publishedCampaign(1, "c1")}}
	repo := newFakeDonationRepo()
	svc := NewService(repo, campRepo, 1, 24)

	req := donation_dto.CreateRequest{
		Amount: 20000, DonorName: "Budi", DonorEmail: "budi@example.com", DonorPhone: "0812",
		IdempotencyKey: "client-generated-key-1",
	}
	first, err := svc.Create(context.Background(), "c1", nil, req)
	if err != nil {
		t.Fatalf("expected first submission to succeed, got error: %v", err)
	}
	second, err := svc.Create(context.Background(), "c1", nil, req)
	if err != nil {
		t.Fatalf("expected duplicate submission to be idempotent (no error), got: %v", err)
	}
	if first.DonationID != second.DonationID {
		t.Fatalf("expected duplicate idempotencyKey to return the same donation, got %d vs %d", first.DonationID, second.DonationID)
	}
	if repo.createCalls != 2 {
		t.Fatalf("expected repo.Create to be attempted twice (second rejected by uniqueness), got %d calls", repo.createCalls)
	}
}
