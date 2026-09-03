package donation_service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"gorm.io/gorm"

	"fsldk-api/base/apperror"
	"fsldk-api/config"
	"fsldk-api/constants"
	"fsldk-api/modules/campaign/campaign_dto"
	"fsldk-api/modules/campaign/campaign_model"
	"fsldk-api/modules/donation/donation_dto"
	"fsldk-api/modules/donation/donation_model"
	"fsldk-api/modules/donation/donation_repository"
	"fsldk-api/modules/jobqueue/jobqueue_dto"
	"fsldk-api/modules/wallet/wallet_dto"
	"fsldk-api/pkg/auditlog"
	"fsldk-api/pkg/bisatopup"
)

// fakeFinanceAuditor adalah implementasi FinanceAuditor no-op — isi audit
// trail tidak relevan diverifikasi di unit test business-logic ini.
type fakeFinanceAuditor struct{}

func (f *fakeFinanceAuditor) LogFinance(ctx context.Context, e auditlog.Entry) {}

type fakeMailer struct {
	receiptCalls []receiptCall
	invoiceCalls int
}

type receiptCall struct {
	ToEmail, CampaignTitle, PublicRef string
}

func (f *fakeMailer) SendDonationReceipt(toEmail, toName, campaignTitle, amount, total, dateStr, publicRef, receiptURL string) error {
	f.receiptCalls = append(f.receiptCalls, receiptCall{ToEmail: toEmail, CampaignTitle: campaignTitle, PublicRef: publicRef})
	return nil
}

func (f *fakeMailer) SendDonationInvoice(toEmail, toName, campaignTitle, amount, qrURL, expiredDateStr string) error {
	f.invoiceCalls++
	return nil
}

// fakeJobEnqueuer adalah implementasi JobEnqueuer no-op — notifikasi WA
// tidak relevan diverifikasi di unit test business-logic ini.
type fakeJobEnqueuer struct{}

func (f *fakeJobEnqueuer) Enqueue(ctx context.Context, in jobqueue_dto.EnqueueInput) (int64, error) {
	return 0, nil
}

// testConfig adalah config.AppConfig minimal yang dibutuhkan donation_service
// untuk pengujian tanpa DB/gateway sungguhan.
func testConfig() config.AppConfig {
	return config.AppConfig{BisatopupQrisMdrPercentCrowdfunding: 1, BisatopupQrisExpiryHoursCrowdfunding: 24}
}

// fakeGateway adalah implementasi bisatopup.Gateway in-memory untuk menguji
// donation_service tanpa panggilan HTTP sungguhan.
type fakeGateway struct {
	createErr error
}

func (f *fakeGateway) CreateQRISTransaction(ctx context.Context, p bisatopup.CreateQRISTransactionParams) (bisatopup.Transaction, error) {
	if f.createErr != nil {
		return bisatopup.Transaction{}, f.createErr
	}
	return bisatopup.Transaction{TransactionID: p.TransactionID, QrCode: "00020101021226670016COM.TEST"}, nil
}
func (f *fakeGateway) DetailTransaction(ctx context.Context, bisabillerID int64) (bisatopup.Transaction, error) {
	return bisatopup.Transaction{}, nil
}
func (f *fakeGateway) ListTransactions(ctx context.Context) ([]bisatopup.Transaction, error) {
	return nil, nil
}
func (f *fakeGateway) InquiryBank(ctx context.Context, bankCode, accountNumber string) (bisatopup.InquiryBankResult, error) {
	return bisatopup.InquiryBankResult{}, nil
}
func (f *fakeGateway) Disburse(ctx context.Context, p bisatopup.DisburseParams) (bisatopup.DisburseResult, error) {
	return bisatopup.DisburseResult{}, nil
}
func (f *fakeGateway) WalletBalance(ctx context.Context) (bisatopup.WalletBalanceResult, error) {
	return bisatopup.WalletBalanceResult{}, nil
}
func (f *fakeGateway) BankList(ctx context.Context) ([]bisatopup.BankListItem, error) {
	return nil, nil
}

// fakeWalletService adalah implementasi wallet_service.Service in-memory —
// hanya CreditDonation yang berperilaku bermakna (mencatat pemanggilan),
// method lain no-op karena tidak dipakai donation_service.
type fakeWalletService struct {
	creditCalls []creditCall
	creditErr   error
}

type creditCall struct {
	CampaignID, DonationID int64
	Amount                 float64
}

func (f *fakeWalletService) CreditDonation(tx *gorm.DB, campaignID, donationID int64, amount float64, note string) error {
	if f.creditErr != nil {
		return f.creditErr
	}
	f.creditCalls = append(f.creditCalls, creditCall{CampaignID: campaignID, DonationID: donationID, Amount: amount})
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

// fakeCampaignRepository adalah implementasi campaign_repository.Repository
// minimal untuk menguji donation_service — hanya FindBySlug yang berperilaku
// bermakna, sisanya no-op karena tidak dipakai skenario uji di file ini.
type fakeCampaignRepository struct {
	bySlug map[string]campaign_model.Campaign
}

func (f *fakeCampaignRepository) List(ctx context.Context, filter campaign_dto.ListFilter) ([]campaign_model.Campaign, int64, error) {
	return nil, 0, nil
}
func (f *fakeCampaignRepository) ListLite(ctx context.Context) ([]campaign_model.Campaign, error) {
	return nil, nil
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
func (f *fakeCampaignRepository) Delete(ctx context.Context, id int64) error {
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

func (f *fakeDonationRepository) UpdateGatewayResult(ctx context.Context, donationID int64, p donation_model.GatewayResultParams) error {
	d := f.byID[donationID]
	d.ExternalTransactionID = sql.NullString{String: p.ExternalTransactionID, Valid: true}
	d.QrPayload = sql.NullString{String: p.QrPayload, Valid: p.QrPayload != ""}
	d.PaymentCode = sql.NullString{String: p.PaymentCode, Valid: p.PaymentCode != ""}
	d.PaymentLink = sql.NullString{String: p.PaymentLink, Valid: p.PaymentLink != ""}
	f.byID[donationID] = d
	return nil
}

func (f *fakeDonationRepository) MarkGatewayFailed(ctx context.Context, donationID int64) error {
	d := f.byID[donationID]
	d.PaymentStatus = constants.DonationStatusFailed
	f.byID[donationID] = d
	return nil
}

func (f *fakeDonationRepository) FindByExternalTransactionIDForUpdate(tx *gorm.DB, externalTransactionID string) (donation_model.Donation, error) {
	for _, d := range f.byID {
		if d.ExternalTransactionID.Valid && d.ExternalTransactionID.String == externalTransactionID {
			return d, nil
		}
	}
	return donation_model.Donation{}, donation_repository.ErrNotFound
}

func (f *fakeDonationRepository) ExpireStalePending(ctx context.Context) (int64, []int64, error) {
	var n int64
	var ids []int64
	for id, d := range f.byID {
		if d.PaymentStatus == constants.DonationStatusPending {
			d.PaymentStatus = constants.DonationStatusExpired
			f.byID[id] = d
			n++
			ids = append(ids, id)
		}
	}
	return n, ids, nil
}

func (f *fakeDonationRepository) UpdateCallbackStatus(tx *gorm.DB, donationID int64, p donation_model.CallbackUpdateParams) error {
	d := f.byID[donationID]
	d.PaymentStatus = p.PaymentStatus
	if p.TotalAmount != nil {
		d.TotalAmount = *p.TotalAmount
	}
	if p.AdminFee != nil {
		d.AdminFee = *p.AdminFee
	}
	f.byID[donationID] = d
	return nil
}

func (f *fakeDonationRepository) CountPaidByCampaign(ctx context.Context, campaignID int64) (int64, error) {
	var n int64
	for _, d := range f.byID {
		if d.CampaignID == campaignID && d.PaymentStatus == constants.DonationStatusPaid {
			n++
		}
	}
	return n, nil
}

func (f *fakeDonationRepository) CountPendingByCampaign(ctx context.Context, campaignID int64) (int64, error) {
	var n int64
	for _, d := range f.byID {
		if d.CampaignID == campaignID && d.PaymentStatus == constants.DonationStatusPending {
			n++
		}
	}
	return n, nil
}

func (f *fakeDonationRepository) AdminCreate(ctx context.Context, p donation_model.AdminCreateParams) (int64, error) {
	f.nextID++
	id := f.nextID
	f.byID[id] = donation_model.Donation{
		DonationID: id, PublicRef: p.PublicRef, CampaignID: p.CampaignID, DonorName: p.DonorName,
		DonorEmail: p.DonorEmail.String, DonorPhone: p.DonorPhone.String, IsAnonymous: p.IsAnonymous,
		Amount: p.Amount, TotalAmount: p.Amount, PaymentStatus: p.PaymentStatus,
		Gateway: constants.DonationGatewayManual, PaymentMethod: p.PaymentMethod,
	}
	return id, nil
}

func (f *fakeDonationRepository) AdminUpdate(ctx context.Context, id int64, p donation_model.AdminUpdateParams) error {
	d := f.byID[id]
	d.DonorName = p.DonorName
	d.Amount = p.Amount
	d.TotalAmount = p.Amount
	d.PaymentStatus = p.PaymentStatus
	d.PaymentMethod = p.PaymentMethod
	f.byID[id] = d
	return nil
}

func (f *fakeDonationRepository) AdminDelete(ctx context.Context, id int64) error {
	delete(f.byID, id)
	return nil
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
	svc := NewService(newFakeDonationRepo(), campRepo, &fakeWalletService{}, &fakeGateway{}, &fakeJobEnqueuer{}, &fakeFinanceAuditor{}, &fakeMailer{}, nil, testConfig())

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
	svc := NewService(newFakeDonationRepo(), campRepo, &fakeWalletService{}, &fakeGateway{}, &fakeJobEnqueuer{}, &fakeFinanceAuditor{}, &fakeMailer{}, nil, testConfig())

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
	svc := NewService(newFakeDonationRepo(), campRepo, &fakeWalletService{}, &fakeGateway{}, &fakeJobEnqueuer{}, &fakeFinanceAuditor{}, &fakeMailer{}, nil, testConfig())

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
	svc := NewService(repo, campRepo, &fakeWalletService{}, &fakeGateway{}, &fakeJobEnqueuer{}, &fakeFinanceAuditor{}, &fakeMailer{}, nil, testConfig())

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

func TestCreate_GatewayRejectionMapsToPaymentFailed(t *testing.T) {
	campRepo := &fakeCampaignRepository{bySlug: map[string]campaign_model.Campaign{"c1": publishedCampaign(1, "c1")}}
	repo := newFakeDonationRepo()
	svc := NewService(repo, campRepo, &fakeWalletService{}, &fakeGateway{createErr: bisatopup.ErrGatewayRejected}, &fakeJobEnqueuer{}, &fakeFinanceAuditor{}, &fakeMailer{}, nil, testConfig())

	_, err := svc.Create(context.Background(), "c1", nil, donation_dto.CreateRequest{
		Amount: 20000, DonorName: "Budi", DonorEmail: "budi@example.com", DonorPhone: "0812",
	})
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodePaymentFailed {
		t.Fatalf("expected PaymentFailed error when gateway rejects the transaction, got %v", err)
	}
	if repo.byID[1].PaymentStatus != constants.DonationStatusFailed {
		t.Fatalf("expected donation to be marked FAILED after gateway rejection, got %v", repo.byID[1].PaymentStatus)
	}
}

func TestCreate_GatewayNetworkFailureMapsToProviderError(t *testing.T) {
	campRepo := &fakeCampaignRepository{bySlug: map[string]campaign_model.Campaign{"c1": publishedCampaign(1, "c1")}}
	repo := newFakeDonationRepo()
	svc := NewService(repo, campRepo, &fakeWalletService{}, &fakeGateway{createErr: errors.New("connection reset")}, &fakeJobEnqueuer{}, &fakeFinanceAuditor{}, &fakeMailer{}, nil, testConfig())

	_, err := svc.Create(context.Background(), "c1", nil, donation_dto.CreateRequest{
		Amount: 20000, DonorName: "Budi", DonorEmail: "budi@example.com", DonorPhone: "0812",
	})
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeProviderError {
		t.Fatalf("expected ProviderError for a generic gateway/network failure, got %v", err)
	}
	if repo.byID[1].PaymentStatus != constants.DonationStatusFailed {
		t.Fatalf("expected donation to be marked FAILED after gateway failure, got %v", repo.byID[1].PaymentStatus)
	}
}

func TestCreate_StoresGatewayQrResultOnSuccess(t *testing.T) {
	campRepo := &fakeCampaignRepository{bySlug: map[string]campaign_model.Campaign{"c1": publishedCampaign(1, "c1")}}
	svc := NewService(newFakeDonationRepo(), campRepo, &fakeWalletService{}, &fakeGateway{}, &fakeJobEnqueuer{}, &fakeFinanceAuditor{}, &fakeMailer{}, nil, testConfig())

	resp, err := svc.Create(context.Background(), "c1", nil, donation_dto.CreateRequest{
		Amount: 20000, DonorName: "Budi", DonorEmail: "budi@example.com", DonorPhone: "0812",
	})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if resp.QrPayload == "" {
		t.Fatal("expected qrPayload to be populated from the gateway response")
	}
}

func TestMapGatewayStatus(t *testing.T) {
	cases := map[int]string{
		1: constants.DonationStatusPending, 2: constants.DonationStatusPending, 13: constants.DonationStatusPending,
		3: constants.DonationStatusPaid, 4: constants.DonationStatusPaid,
		5:  constants.DonationStatusCancelled,
		6:  constants.DonationStatusRefunded,
		14: constants.DonationStatusFailed,
		99: constants.DonationStatusPending, // status_id tak dikenal jatuh ke default PENDING
	}
	for statusID, want := range cases {
		if got := mapGatewayStatus(statusID); got != want {
			t.Errorf("mapGatewayStatus(%d) = %s, want %s", statusID, got, want)
		}
	}
}

func TestIsFinalDonationStatus(t *testing.T) {
	final := []string{constants.DonationStatusPaid, constants.DonationStatusFailed, constants.DonationStatusCancelled, constants.DonationStatusRefunded, constants.DonationStatusAmountMismatch}
	for _, s := range final {
		if !isFinalDonationStatus(s) {
			t.Errorf("expected %s to be a final status", s)
		}
	}
	nonFinal := []string{constants.DonationStatusPending, constants.DonationStatusExpired}
	for _, s := range nonFinal {
		if isFinalDonationStatus(s) {
			t.Errorf("expected %s to NOT be final — late callback recovery must still be able to override it", s)
		}
	}
}

// Kriteria & catatan cakupan lihat komentar isTestCallback — sengaja BUKAN
// digerbang oleh environment (dev/live) lagi: dashboard Bisatopup mengirim
// ping "Test" yang sama persis baik saat URL callback yang diuji itu live
// maupun dev, jadi harus dikenali di kedua env (bug sebelumnya: hanya
// dikenali di dev, jadi ping ke URL production selalu gagal).
func TestIsTestCallback(t *testing.T) {
	if !isTestCallback(donation_dto.CallbackRequest{Signature: "testing"}) {
		t.Fatal("signature=testing must be recognised as a test callback in any environment")
	}
	if !isTestCallback(donation_dto.CallbackRequest{TransactionID: "TEST-abc123", Signature: "abc"}) {
		t.Fatal("transaction_id starting with TEST- must be recognised as a test callback")
	}
	if !isTestCallback(donation_dto.CallbackRequest{}) {
		t.Fatal("empty transaction_id (general connectivity ping) must be recognised as a test callback")
	}
	if isTestCallback(donation_dto.CallbackRequest{TransactionID: "REAL-TXN-001", Signature: "abc123"}) {
		t.Fatal("a real transaction_id with a real-looking signature must not be treated as a test callback")
	}
}

func TestTruncateNoEllipsis(t *testing.T) {
	if got := truncateNoEllipsis("Donasi Kampanye Sangat Panjang Sekali Sekali Sekali", 10); got != "Donasi Kam" {
		t.Fatalf("truncateNoEllipsis = %q, want exactly 10 chars with no suffix", got)
	}
	if got := truncateNoEllipsis("short", 10); got != "short" {
		t.Fatalf("truncateNoEllipsis should not modify strings shorter than max, got %q", got)
	}
}

// TestNotifyDonationPaid_SendsReceiptEmailWhenDonorEmailPresent menutupi
// fitur baru (revisi 2026-08-30) — konfirmasi donasi lewat email, dipicu
// bersamaan dengan notifikasi WA saat donasi PAID, gated pada DonorEmail
// tidak kosong (pola sama ldksyahid-app Celengan Syahid).
func TestNotifyDonationPaid_SendsReceiptEmailWhenDonorEmailPresent(t *testing.T) {
	mail := &fakeMailer{}
	svc := &ServiceImpl{campaignRepo: &fakeCampaignRepository{}, jobs: &fakeJobEnqueuer{}, mail: mail, cfg: testConfig()}

	svc.notifyDonationPaid(context.Background(), donation_model.Donation{
		DonationID: 1, DonorName: "Budi", DonorEmail: "budi@example.com", DonorPhone: "081234567890",
		CampaignTitle: "Bantu Sesama", PublicRef: "abc-123", Amount: 100000, TotalAmount: 101000,
	})

	if len(mail.receiptCalls) != 1 {
		t.Fatalf("expected 1 receipt email to be sent, got %d", len(mail.receiptCalls))
	}
	if mail.receiptCalls[0].ToEmail != "budi@example.com" || mail.receiptCalls[0].PublicRef != "abc-123" {
		t.Fatalf("unexpected receipt email call: %+v", mail.receiptCalls[0])
	}
}

func TestNotifyDonationPaid_SkipsReceiptEmailWhenDonorEmailEmpty(t *testing.T) {
	mail := &fakeMailer{}
	svc := &ServiceImpl{campaignRepo: &fakeCampaignRepository{}, jobs: &fakeJobEnqueuer{}, mail: mail, cfg: testConfig()}

	svc.notifyDonationPaid(context.Background(), donation_model.Donation{
		DonationID: 1, DonorName: "Anonim", DonorEmail: "", CampaignTitle: "Bantu Sesama", PublicRef: "abc-123",
	})

	if len(mail.receiptCalls) != 0 {
		t.Fatalf("expected no receipt email when DonorEmail is empty, got %d", len(mail.receiptCalls))
	}
}

func TestExpireStale_OnlyExpiresPendingDonations(t *testing.T) {
	repo := newFakeDonationRepo()
	repo.byID[1] = donation_model.Donation{DonationID: 1, PaymentStatus: constants.DonationStatusPending}
	repo.byID[2] = donation_model.Donation{DonationID: 2, PaymentStatus: constants.DonationStatusPending}
	repo.byID[3] = donation_model.Donation{DonationID: 3, PaymentStatus: constants.DonationStatusPaid}

	campRepo := &fakeCampaignRepository{bySlug: map[string]campaign_model.Campaign{}}
	svc := NewService(repo, campRepo, &fakeWalletService{}, &fakeGateway{}, &fakeJobEnqueuer{}, &fakeFinanceAuditor{}, &fakeMailer{}, nil, testConfig())

	n, err := svc.ExpireStale(context.Background())
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 donations expired, got %d", n)
	}
	if repo.byID[3].PaymentStatus != constants.DonationStatusPaid {
		t.Fatal("expected PAID donation to be untouched by ExpireStale")
	}
}
