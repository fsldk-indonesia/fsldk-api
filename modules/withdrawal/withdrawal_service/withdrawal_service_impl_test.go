package withdrawal_service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"fsldk-api/base/apperror"
	"fsldk-api/config"
	"fsldk-api/constants"
	"fsldk-api/modules/campaign/campaign_dto"
	"fsldk-api/modules/campaign/campaign_model"
	"fsldk-api/modules/wallet/wallet_dto"
	"fsldk-api/modules/withdrawal/withdrawal_dto"
	"fsldk-api/modules/withdrawal/withdrawal_model"
	"fsldk-api/pkg/bisatopup"
)

// -- fakes --

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

func ownedCampaign(id, ownerUserID int64) campaign_model.Campaign {
	return campaign_model.Campaign{
		CampaignID: id, OwnerUserID: ownerUserID, Title: "Campaign Uji",
		BeneficiaryBankCode: "bca", BeneficiaryAccountNumber: "12345", BeneficiaryAccountHolder: "Budi",
	}
}

type fakeWithdrawalRepository struct {
	nextID       int64
	byID         map[int64]withdrawal_model.Withdrawal
	byIdempotent map[string]int64
	activeCount  int64
	staleRows    []withdrawal_model.Withdrawal
}

func newFakeWithdrawalRepo() *fakeWithdrawalRepository {
	return &fakeWithdrawalRepository{byID: map[int64]withdrawal_model.Withdrawal{}, byIdempotent: map[string]int64{}}
}

func (f *fakeWithdrawalRepository) Create(tx *gorm.DB, p withdrawal_model.CreateParams) (int64, error) {
	f.nextID++
	id := f.nextID
	f.byID[id] = withdrawal_model.Withdrawal{
		WithdrawalID: id, WithdrawalRef: p.WithdrawalRef, CampaignID: p.CampaignID,
		RequestedByUserID: p.RequestedByUserID, Amount: p.Amount, Fee: p.Fee, NetAmount: p.NetAmount,
		Status: constants.WithdrawalStatusSecurityCheck, IdempotencyKey: p.IdempotencyKey,
	}
	f.byIdempotent[p.IdempotencyKey] = id
	return id, nil
}
func (f *fakeWithdrawalRepository) FindByID(ctx context.Context, id int64) (withdrawal_model.Withdrawal, error) {
	w, ok := f.byID[id]
	if !ok {
		return withdrawal_model.Withdrawal{}, errors.New("not found")
	}
	return w, nil
}
func (f *fakeWithdrawalRepository) FindByIdempotencyKey(ctx context.Context, key string) (withdrawal_model.Withdrawal, error) {
	id, ok := f.byIdempotent[key]
	if !ok {
		return withdrawal_model.Withdrawal{}, errors.New("not found")
	}
	return f.byID[id], nil
}
func (f *fakeWithdrawalRepository) CountNonFinalByCampaign(ctx context.Context, campaignID int64) (int64, error) {
	return f.activeCount, nil
}
func (f *fakeWithdrawalRepository) CountNonFinalByCampaignForUpdate(tx *gorm.DB, campaignID int64) (int64, error) {
	return f.activeCount, nil
}
func (f *fakeWithdrawalRepository) List(ctx context.Context, filt withdrawal_dto.ListFilter) ([]withdrawal_model.Withdrawal, int64, error) {
	return nil, 0, nil
}
func (f *fakeWithdrawalRepository) UpdateStatus(tx *gorm.DB, id int64, status string, p withdrawal_model.StatusUpdateParams) error {
	w := f.byID[id]
	w.Status = status
	f.byID[id] = w
	return nil
}
func (f *fakeWithdrawalRepository) FindByRefForUpdate(tx *gorm.DB, ref string) (withdrawal_model.Withdrawal, error) {
	for _, w := range f.byID {
		if w.WithdrawalRef == ref {
			return w, nil
		}
	}
	return withdrawal_model.Withdrawal{}, errors.New("not found")
}
func (f *fakeWithdrawalRepository) FindStaleProcessing(ctx context.Context, olderThan time.Time) ([]withdrawal_model.Withdrawal, error) {
	return f.staleRows, nil
}

type fakeWalletService struct{}

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
	return wallet_dto.BalanceResponse{}, nil
}
func (f *fakeWalletService) ListLedger(ctx context.Context, campaignID int64, filter wallet_dto.LedgerListFilter) ([]wallet_dto.LedgerListItem, int64, error) {
	return nil, 0, nil
}
func (f *fakeWalletService) GetBalanceForOwner(ctx context.Context, campaignID, ownerUserID int64) (wallet_dto.BalanceResponse, error) {
	return wallet_dto.BalanceResponse{}, nil
}
func (f *fakeWalletService) ListLedgerForOwner(ctx context.Context, campaignID, ownerUserID int64, filter wallet_dto.LedgerListFilter) ([]wallet_dto.LedgerListItem, int64, error) {
	return nil, 0, nil
}

type fakeGateway struct {
	inquiryResult bisatopup.InquiryBankResult
	inquiryErr    error
	bankList      []bisatopup.BankListItem
}

func (f *fakeGateway) CreateQRISTransaction(ctx context.Context, p bisatopup.CreateQRISTransactionParams) (bisatopup.Transaction, error) {
	return bisatopup.Transaction{}, nil
}
func (f *fakeGateway) DetailTransaction(ctx context.Context, bisabillerID int64) (bisatopup.Transaction, error) {
	return bisatopup.Transaction{}, nil
}
func (f *fakeGateway) ListTransactions(ctx context.Context) ([]bisatopup.Transaction, error) {
	return nil, nil
}
func (f *fakeGateway) InquiryBank(ctx context.Context, bankCode, accountNumber string) (bisatopup.InquiryBankResult, error) {
	if f.inquiryErr != nil {
		return bisatopup.InquiryBankResult{}, f.inquiryErr
	}
	return f.inquiryResult, nil
}
func (f *fakeGateway) Disburse(ctx context.Context, p bisatopup.DisburseParams) (bisatopup.DisburseResult, error) {
	return bisatopup.DisburseResult{}, nil
}
func (f *fakeGateway) WalletBalance(ctx context.Context) (bisatopup.WalletBalanceResult, error) {
	return bisatopup.WalletBalanceResult{}, nil
}
func (f *fakeGateway) BankList(ctx context.Context) ([]bisatopup.BankListItem, error) {
	return f.bankList, nil
}

func successfulInquiry() bisatopup.InquiryBankResult {
	return bisatopup.InquiryBankResult{Status: "SUCCESS", AccountHolder: "Budi", Fee: "3000"}
}

func testConfig() config.AppConfig { return config.AppConfig{} }

// -- Request() tests: only branches that return before opening a real DB
// transaction are unit-tested here (db=nil is safe only for those) — the
// successful-insert path is verified end-to-end against real MySQL in the
// integration smoke test, same pattern as donation_service/wallet_service.

func TestRequest_RejectsNonOwner(t *testing.T) {
	campRepo := &fakeCampaignRepository{byID: map[int64]campaign_model.Campaign{1: ownedCampaign(1, 10)}}
	svc := NewService(newFakeWithdrawalRepo(), campRepo, &fakeWalletService{}, &fakeGateway{}, nil, testConfig())

	_, err := svc.Request(context.Background(), 1, 99, withdrawal_dto.CreateRequest{Amount: 100000})
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeNotFound {
		t.Fatalf("expected NotFound (IDOR-safe) for non-owner, got %v", err)
	}
}

func TestRequest_RejectsWhenActiveWithdrawalExists(t *testing.T) {
	campRepo := &fakeCampaignRepository{byID: map[int64]campaign_model.Campaign{1: ownedCampaign(1, 10)}}
	repo := newFakeWithdrawalRepo()
	repo.activeCount = 1
	svc := NewService(repo, campRepo, &fakeWalletService{}, &fakeGateway{}, nil, testConfig())

	_, err := svc.Request(context.Background(), 1, 10, withdrawal_dto.CreateRequest{Amount: 100000})
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeConflict {
		t.Fatalf("expected Conflict when an active withdrawal already exists, got %v", err)
	}
}

func TestRequest_RejectsInvalidBeneficiary(t *testing.T) {
	campRepo := &fakeCampaignRepository{byID: map[int64]campaign_model.Campaign{1: ownedCampaign(1, 10)}}
	svc := NewService(newFakeWithdrawalRepo(), campRepo, &fakeWalletService{}, &fakeGateway{inquiryErr: bisatopup.ErrGatewayRejected}, nil, testConfig())

	_, err := svc.Request(context.Background(), 1, 10, withdrawal_dto.CreateRequest{Amount: 100000})
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeUnprocessable {
		t.Fatalf("expected Unprocessable when beneficiary inquiry fails, got %v", err)
	}
}

func TestRequest_RejectsWhenNetAmountNotPositive(t *testing.T) {
	campRepo := &fakeCampaignRepository{byID: map[int64]campaign_model.Campaign{1: ownedCampaign(1, 10)}}
	gw := &fakeGateway{inquiryResult: bisatopup.InquiryBankResult{Status: "SUCCESS", Fee: "999999"}}
	svc := NewService(newFakeWithdrawalRepo(), campRepo, &fakeWalletService{}, gw, nil, testConfig())

	_, err := svc.Request(context.Background(), 1, 10, withdrawal_dto.CreateRequest{Amount: 50000})
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeValidationError {
		t.Fatalf("expected BadRequest when fee exceeds amount, got %v", err)
	}
}

// -- Approve() tests --

func TestApprove_RejectsWrongStatus(t *testing.T) {
	repo := newFakeWithdrawalRepo()
	repo.byID[1] = withdrawal_model.Withdrawal{WithdrawalID: 1, Status: constants.WithdrawalStatusSecurityCheck, RequestedByUserID: 10}
	svc := NewService(repo, &fakeCampaignRepository{}, &fakeWalletService{}, &fakeGateway{}, nil, testConfig())

	_, err := svc.Approve(context.Background(), 1, 20)
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeInvalidStatusTransition {
		t.Fatalf("expected InvalidStatusTransition, got %v", err)
	}
}

func TestApprove_RejectsSameActorAsRequester(t *testing.T) {
	repo := newFakeWithdrawalRepo()
	repo.byID[1] = withdrawal_model.Withdrawal{WithdrawalID: 1, Status: constants.WithdrawalStatusPendingApproval, RequestedByUserID: 10}
	svc := NewService(repo, &fakeCampaignRepository{}, &fakeWalletService{}, &fakeGateway{}, nil, testConfig())

	_, err := svc.Approve(context.Background(), 1, 10)
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeForbidden {
		t.Fatalf("expected Forbidden when approver == requester (maker-checker), got %v", err)
	}
}

// -- Reject() tests --

func TestReject_RequiresReason(t *testing.T) {
	repo := newFakeWithdrawalRepo()
	repo.byID[1] = withdrawal_model.Withdrawal{WithdrawalID: 1, Status: constants.WithdrawalStatusPendingApproval}
	svc := NewService(repo, &fakeCampaignRepository{}, &fakeWalletService{}, &fakeGateway{}, nil, testConfig())

	err := svc.Reject(context.Background(), 1, 20, "   ")
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeValidationError {
		t.Fatalf("expected BadRequest for empty reason, got %v", err)
	}
}

func TestReject_RejectsWrongStatus(t *testing.T) {
	repo := newFakeWithdrawalRepo()
	repo.byID[1] = withdrawal_model.Withdrawal{WithdrawalID: 1, Status: constants.WithdrawalStatusApproved}
	svc := NewService(repo, &fakeCampaignRepository{}, &fakeWalletService{}, &fakeGateway{}, nil, testConfig())

	err := svc.Reject(context.Background(), 1, 20, "alasan valid")
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeInvalidStatusTransition {
		t.Fatalf("expected InvalidStatusTransition, got %v", err)
	}
}

// -- Cancel() tests --

func TestCancel_RejectsNonRequester(t *testing.T) {
	repo := newFakeWithdrawalRepo()
	repo.byID[1] = withdrawal_model.Withdrawal{WithdrawalID: 1, RequestedByUserID: 10, Status: constants.WithdrawalStatusSecurityCheck}
	svc := NewService(repo, &fakeCampaignRepository{}, &fakeWalletService{}, &fakeGateway{}, nil, testConfig())

	err := svc.Cancel(context.Background(), 1, 99)
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeNotFound {
		t.Fatalf("expected NotFound (IDOR-safe) for non-requester cancel, got %v", err)
	}
}

func TestCancel_RejectsWrongStatus(t *testing.T) {
	repo := newFakeWithdrawalRepo()
	repo.byID[1] = withdrawal_model.Withdrawal{WithdrawalID: 1, RequestedByUserID: 10, Status: constants.WithdrawalStatusProcessing}
	svc := NewService(repo, &fakeCampaignRepository{}, &fakeWalletService{}, &fakeGateway{}, nil, testConfig())

	err := svc.Cancel(context.Background(), 1, 10)
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeInvalidStatusTransition {
		t.Fatalf("expected InvalidStatusTransition for cancelling a PROCESSING withdrawal, got %v", err)
	}
}

// -- Process() tests --

func TestProcess_RejectsWrongStatus(t *testing.T) {
	repo := newFakeWithdrawalRepo()
	repo.byID[1] = withdrawal_model.Withdrawal{WithdrawalID: 1, Status: constants.WithdrawalStatusPendingApproval}
	svc := NewService(repo, &fakeCampaignRepository{}, &fakeWalletService{}, &fakeGateway{}, nil, testConfig())

	_, err := svc.Process(context.Background(), 1, 20)
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeInvalidStatusTransition {
		t.Fatalf("expected InvalidStatusTransition (must be APPROVED first), got %v", err)
	}
}

// -- Inquiry / ListBanks --

func TestInquiry_ReturnsAccountHolderAndFee(t *testing.T) {
	gw := &fakeGateway{inquiryResult: successfulInquiry()}
	svc := NewService(newFakeWithdrawalRepo(), &fakeCampaignRepository{}, &fakeWalletService{}, gw, nil, testConfig())

	resp, err := svc.Inquiry(context.Background(), withdrawal_dto.InquiryRequest{BankCode: "bca", AccountNumber: "12345"})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if resp.AccountHolder != "Budi" || resp.Fee != 3000 {
		t.Fatalf("unexpected inquiry response: %+v", resp)
	}
}

func TestInquiry_GatewayRejectionMapsToUnprocessable(t *testing.T) {
	gw := &fakeGateway{inquiryErr: bisatopup.ErrGatewayRejected}
	svc := NewService(newFakeWithdrawalRepo(), &fakeCampaignRepository{}, &fakeWalletService{}, gw, nil, testConfig())

	_, err := svc.Inquiry(context.Background(), withdrawal_dto.InquiryRequest{BankCode: "bca", AccountNumber: "12345"})
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeUnprocessable {
		t.Fatalf("expected Unprocessable, got %v", err)
	}
}

func TestListBanks_ReturnsMappedItems(t *testing.T) {
	gw := &fakeGateway{bankList: []bisatopup.BankListItem{{BankCode: "bca", Name: "BCA", Fee: 3000, Status: "OPERATIONAL"}}}
	svc := NewService(newFakeWithdrawalRepo(), &fakeCampaignRepository{}, &fakeWalletService{}, gw, nil, testConfig())

	banks, err := svc.ListBanks(context.Background())
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if len(banks) != 1 || banks[0].BankCode != "bca" || banks[0].Fee != 3000 {
		t.Fatalf("unexpected bank list: %+v", banks)
	}
}

// -- ReconcileStaleProcessing --

func TestReconcileStaleProcessing_ReturnsStaleRows(t *testing.T) {
	repo := newFakeWithdrawalRepo()
	repo.staleRows = []withdrawal_model.Withdrawal{{WithdrawalID: 1, Status: constants.WithdrawalStatusProcessing}}
	svc := NewService(repo, &fakeCampaignRepository{}, &fakeWalletService{}, &fakeGateway{}, nil, testConfig())

	rows, err := svc.ReconcileStaleProcessing(context.Background())
	if err != nil || len(rows) != 1 {
		t.Fatalf("expected 1 stale row, got %d rows err=%v", len(rows), err)
	}
}

// -- pure helper functions --

func TestMapDisbursementStatus(t *testing.T) {
	cases := map[int]string{
		3: constants.WithdrawalStatusSuccess, 4: constants.WithdrawalStatusSuccess,
		5: constants.WithdrawalStatusFailed, 14: constants.WithdrawalStatusFailed,
		1: "", 2: "", 99: "",
	}
	for statusID, want := range cases {
		if got := mapDisbursementStatus(statusID); got != want {
			t.Errorf("mapDisbursementStatus(%d) = %q, want %q", statusID, got, want)
		}
	}
}

func TestIsFinalWithdrawalStatus(t *testing.T) {
	final := []string{constants.WithdrawalStatusSuccess, constants.WithdrawalStatusFailed, constants.WithdrawalStatusRejected, constants.WithdrawalStatusCancelled, constants.WithdrawalStatusReversed}
	for _, s := range final {
		if !isFinalWithdrawalStatus(s) {
			t.Errorf("expected %s to be final", s)
		}
	}
	nonFinal := []string{constants.WithdrawalStatusRequested, constants.WithdrawalStatusSecurityCheck, constants.WithdrawalStatusPendingApproval, constants.WithdrawalStatusApproved, constants.WithdrawalStatusProcessing}
	for _, s := range nonFinal {
		if isFinalWithdrawalStatus(s) {
			t.Errorf("expected %s to NOT be final", s)
		}
	}
}

func TestIsCancellableStatus(t *testing.T) {
	cancellable := []string{constants.WithdrawalStatusRequested, constants.WithdrawalStatusSecurityCheck, constants.WithdrawalStatusPendingApproval}
	for _, s := range cancellable {
		if !isCancellableStatus(s) {
			t.Errorf("expected %s to be cancellable", s)
		}
	}
	notCancellable := []string{constants.WithdrawalStatusApproved, constants.WithdrawalStatusProcessing, constants.WithdrawalStatusSuccess}
	for _, s := range notCancellable {
		if isCancellableStatus(s) {
			t.Errorf("expected %s to NOT be cancellable", s)
		}
	}
}

func TestFallbackIdempotencyKey_SameInputsSameWindow_SameKey(t *testing.T) {
	now := time.Now()
	k1 := fallbackIdempotencyKey(1, 100000, now)
	k2 := fallbackIdempotencyKey(1, 100000, now)
	if k1 != k2 {
		t.Fatal("expected identical inputs in the same window to produce the same key")
	}
}

func TestFallbackIdempotencyKey_DifferentCampaign_DifferentKey(t *testing.T) {
	now := time.Now()
	k1 := fallbackIdempotencyKey(1, 100000, now)
	k2 := fallbackIdempotencyKey(2, 100000, now)
	if k1 == k2 {
		t.Fatal("expected different campaigns to produce different keys")
	}
}

func TestTruncateNoEllipsis(t *testing.T) {
	if got := truncateNoEllipsis("Penarikan saldo campaign yang sangat panjang", 10); got != "Penarikan " {
		t.Fatalf("truncateNoEllipsis = %q, want exactly 10 chars", got)
	}
}
