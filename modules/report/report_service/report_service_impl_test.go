package report_service

import (
	"context"
	"testing"
	"time"

	"fsldk-api/base/dto"
	"fsldk-api/config"
	"fsldk-api/constants"
	"fsldk-api/modules/report/report_dto"
	"fsldk-api/pkg/bisatopup"
)

// fakeReportRepository adalah implementasi report_repository.Repository
// in-memory untuk menguji business logic report_service (formula balance
// report & threshold rekonsiliasi) tanpa DB sungguhan — nilai balik diatur
// langsung lewat field, bukan disimulasikan dari data mentah.
type fakeReportRepository struct {
	balanceBefore float64
	balanceAsOf   float64
	ledgerSums    map[string]float64 // entryType -> amount, dipakai LedgerSumByType & LedgerTotalByType
	withdrawnSum  float64
	feeSum        float64

	donationPaidCount  int64
	donationPaidAmount float64
	withdrawalCount    int64
	withdrawalAmount   float64

	createdSnapshot report_dto.ReconciliationSnapshotParams
}

func (f *fakeReportRepository) SubmissionRows(ctx context.Context, formID int64, status string, organizationIDs []int64) ([]report_dto.SubmissionRow, error) {
	return nil, nil
}
func (f *fakeReportRepository) BalanceBefore(ctx context.Context, campaignID int64, before time.Time) (float64, error) {
	return f.balanceBefore, nil
}
func (f *fakeReportRepository) BalanceAsOf(ctx context.Context, campaignID int64, asOf time.Time) (float64, error) {
	return f.balanceAsOf, nil
}
func (f *fakeReportRepository) LedgerSumByType(ctx context.Context, campaignID int64, entryType string, from, to time.Time) (float64, error) {
	return f.ledgerSums[entryType], nil
}
func (f *fakeReportRepository) WithdrawalSuccessSum(ctx context.Context, campaignID int64, from, to time.Time) (float64, error) {
	return f.withdrawnSum, nil
}
func (f *fakeReportRepository) FeeSum(ctx context.Context, campaignID int64, from, to time.Time) (float64, error) {
	return f.feeSum, nil
}
func (f *fakeReportRepository) CampaignReportRows(ctx context.Context, filt report_dto.KantongAmalReportFilter) ([]report_dto.CampaignReportRow, int64, error) {
	return nil, 0, nil
}
func (f *fakeReportRepository) DonationReportRows(ctx context.Context, filt report_dto.KantongAmalReportFilter) ([]report_dto.DonationReportRow, int64, error) {
	return nil, 0, nil
}
func (f *fakeReportRepository) WithdrawalReportRows(ctx context.Context, filt report_dto.KantongAmalReportFilter) ([]report_dto.WithdrawalReportRow, int64, error) {
	return nil, 0, nil
}
func (f *fakeReportRepository) WithdrawalStatusFunnel(ctx context.Context, campaignID int64) ([]report_dto.WithdrawalStatusFunnel, error) {
	return nil, nil
}
func (f *fakeReportRepository) DonationPaidTotals(ctx context.Context) (int64, float64, error) {
	return f.donationPaidCount, f.donationPaidAmount, nil
}
func (f *fakeReportRepository) WithdrawalSuccessTotals(ctx context.Context) (int64, float64, error) {
	return f.withdrawalCount, f.withdrawalAmount, nil
}
func (f *fakeReportRepository) LedgerTotalByType(ctx context.Context, entryType string) (float64, error) {
	return f.ledgerSums[entryType], nil
}
func (f *fakeReportRepository) CreateReconciliationSnapshot(ctx context.Context, p report_dto.ReconciliationSnapshotParams) (int64, error) {
	f.createdSnapshot = p
	return 1, nil
}
func (f *fakeReportRepository) ListReconciliationSnapshots(ctx context.Context, q dto.ListQuery) ([]report_dto.ReconciliationSnapshotResponse, int64, error) {
	return nil, 0, nil
}
func (f *fakeReportRepository) ListFinanceAuditLog(ctx context.Context, filt report_dto.FinanceAuditLogFilter) ([]report_dto.FinanceAuditLogItem, int64, error) {
	return nil, 0, nil
}

// fakeGateway adalah implementasi bisatopup.Gateway minimal — hanya
// WalletBalance yang berperilaku bermakna (dipakai RunReconciliation).
type fakeGateway struct {
	walletBalance int64
	walletErr     error
}

func (f *fakeGateway) CreateQRISTransaction(ctx context.Context, p bisatopup.CreateQRISTransactionParams) (bisatopup.Transaction, error) {
	return bisatopup.Transaction{}, nil
}
func (f *fakeGateway) DetailTransaction(ctx context.Context, id int64) (bisatopup.Transaction, error) {
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
	if f.walletErr != nil {
		return bisatopup.WalletBalanceResult{}, f.walletErr
	}
	return bisatopup.WalletBalanceResult{Amount: f.walletBalance}, nil
}
func (f *fakeGateway) BankList(ctx context.Context) ([]bisatopup.BankListItem, error) {
	return nil, nil
}

// audit di-nil-kan di seluruh test file ini — report_service memakai
// concrete *auditlog.Logger, dan method yang diuji (GetBalanceReport/
// RunReconciliation) tidak pernah memanggil LogExport/LogFinance.

func testReportConfig(settlementMinutes int) config.AppConfig {
	return config.AppConfig{BisatopupSettlementMinutesCrowdfunding: settlementMinutes}
}

func TestGetBalanceReport_BalancedWhenArithmeticMatches(t *testing.T) {
	repo := &fakeReportRepository{
		balanceBefore: 100_000, balanceAsOf: 250_000,
		ledgerSums: map[string]float64{
			constants.LedgerEntryDonationCredit:   200_000,
			constants.LedgerEntryRefundDebit:      0,
			constants.LedgerEntryAdjustmentCredit: 0,
			constants.LedgerEntryAdjustmentDebit:  0,
		},
		withdrawnSum: 50_000,
	}
	svc := NewService(repo, nil, nil, nil, &fakeGateway{}, testReportConfig(15))

	resp, err := svc.GetBalanceReport(context.Background(), report_dto.BalanceReportFilter{
		From: time.Now().AddDate(0, 0, -7), To: time.Now(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// opening(100k) + incoming(200k) - outgoing(50k) - refund(0) + adjustment(0) = 250k == closing(250k)
	if !resp.IsBalanced {
		t.Fatalf("expected balanced report, got expectedClosing=%.2f closing=%.2f", resp.ExpectedClosing, resp.ClosingBalance)
	}
}

func TestGetBalanceReport_FlagsMismatchAsNotBalanced(t *testing.T) {
	repo := &fakeReportRepository{
		balanceBefore: 100_000, balanceAsOf: 999_000, // closing sengaja tidak nyambung ke arithmetic
		ledgerSums:   map[string]float64{constants.LedgerEntryDonationCredit: 200_000},
		withdrawnSum: 50_000,
	}
	svc := NewService(repo, nil, nil, nil, &fakeGateway{}, testReportConfig(15))

	resp, err := svc.GetBalanceReport(context.Background(), report_dto.BalanceReportFilter{From: time.Now().AddDate(0, 0, -7), To: time.Now()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.IsBalanced {
		t.Fatalf("expected mismatch to be flagged as not balanced, got isBalanced=true")
	}
}

func TestRunReconciliation_NoAnomalyWithinThreshold(t *testing.T) {
	repo := &fakeReportRepository{
		donationPaidCount: 5, donationPaidAmount: 500_000,
		withdrawalCount: 1, withdrawalAmount: 100_000,
		balanceAsOf: 400_000,
		ledgerSums:  map[string]float64{constants.LedgerEntryDonationCredit: 0},
	}
	// gateway balance selisih 30rb dari expected (400rb) — di bawah ambang Rp50rb (OQ-22).
	gw := &fakeGateway{walletBalance: 430_000}
	svc := NewService(repo, nil, nil, nil, gw, testReportConfig(15))

	snap, err := svc.RunReconciliation(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.HasAnomaly {
		t.Fatalf("expected discrepancy within Rp50.000 threshold to not be flagged, discrepancy=%.2f", snap.DiscrepancyAmount)
	}
}

func TestRunReconciliation_FlagsAnomalyBeyondThreshold(t *testing.T) {
	repo := &fakeReportRepository{
		donationPaidCount: 5, donationPaidAmount: 500_000,
		withdrawalCount: 1, withdrawalAmount: 100_000,
		balanceAsOf: 400_000,
		ledgerSums:  map[string]float64{constants.LedgerEntryDonationCredit: 0}, // tidak ada donasi baru masuk jendela settlement
	}
	// selisih 500rb, jauh di atas ambang Rp50rb + tidak ada recently-paid donation menutupinya.
	gw := &fakeGateway{walletBalance: 900_000}
	svc := NewService(repo, nil, nil, nil, gw, testReportConfig(15))

	snap, err := svc.RunReconciliation(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !snap.HasAnomaly {
		t.Fatalf("expected large discrepancy to be flagged as anomaly, discrepancy=%.2f", snap.DiscrepancyAmount)
	}
}

func TestRunReconciliation_GatewayErrorDoesNotFlagAnomaly(t *testing.T) {
	repo := &fakeReportRepository{balanceAsOf: 400_000}
	gw := &fakeGateway{walletErr: bisatopup.ErrGatewayRejected}
	svc := NewService(repo, nil, nil, nil, gw, testReportConfig(15))

	snap, err := svc.RunReconciliation(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Gateway tidak terjangkau bukan berarti ada anomali finansial — hanya
	// berarti pemeriksaan tidak bisa dilakukan kali ini (dicatat gatewayError).
	if snap.HasAnomaly {
		t.Fatalf("expected gateway error to NOT be treated as a financial anomaly, got hasAnomaly=true")
	}
}
