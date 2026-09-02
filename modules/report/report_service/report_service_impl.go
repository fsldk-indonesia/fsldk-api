package report_service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"time"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/config"
	"fsldk-api/constants"
	"fsldk-api/modules/report/report_dto"
	"fsldk-api/modules/report/report_repository"
	"fsldk-api/modules/submission_form/submission_form_repository"
	"fsldk-api/pkg/auditlog"
	"fsldk-api/pkg/bisatopup"

	"github.com/xuri/excelize/v2"
)

// reconciliationDiscrepancyThreshold adalah ambang selisih ledger vs wallet
// gateway yang dianggap wajar sebelum di-flag anomali — keputusan final
// OQ-22 (reuse ldksyahid-app): Rp50.000.
const reconciliationDiscrepancyThreshold = 50_000

// reconciliationCheckInterval — job terjadwal internal `finance.daily_reconciliation`
// (§13.4 techspec: goroutine time.Ticker langsung, bukan event-driven).
const reconciliationCheckInterval = 24 * time.Hour

var reportColumns = []string{"Nama Organisasi", "Provinsi", "Kota/Kabupaten", "Status", "Level", "Tanggal Submit", "Terakhir Diperbarui"}

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct {
	repo     report_repository.Repository
	formRepo submission_form_repository.Repository
	orgScope OrgScopeResolver
	audit    *auditlog.Logger
	gateway  bisatopup.Gateway
	cfg      config.AppConfig
}

// NewService membuat Service report.
func NewService(repo report_repository.Repository, formRepo submission_form_repository.Repository, orgScope OrgScopeResolver, audit *auditlog.Logger, gateway bisatopup.Gateway, cfg config.AppConfig) Service {
	return &ServiceImpl{repo: repo, formRepo: formRepo, orgScope: orgScope, audit: audit, gateway: gateway, cfg: cfg}
}

func rowValues(row report_dto.SubmissionRow) []string {
	fmtDate := func(t time.Time, valid bool) string {
		if !valid {
			return ""
		}
		return t.Format("2006-01-02 15:04")
	}
	return []string{
		row.OrganizationName,
		row.ProvinceName.String,
		row.CityName.String,
		row.Status,
		row.LevelLabel.String,
		fmtDate(row.SubmittedDate.Time, row.SubmittedDate.Valid),
		fmtDate(row.LastUpdatedDate.Time, row.LastUpdatedDate.Valid),
	}
}

func toCSV(rows []report_dto.SubmissionRow) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write(reportColumns); err != nil {
		return nil, err
	}
	for _, row := range rows {
		if err := w.Write(rowValues(row)); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func toXLSX(rows []report_dto.SubmissionRow) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	const sheet = "Laporan"
	f.SetSheetName("Sheet1", sheet)
	for i, col := range reportColumns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, col)
	}
	for r, row := range rows {
		for c, v := range rowValues(row) {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			f.SetCellValue(sheet, cell, v)
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *ServiceImpl) ExportSubmissions(ctx context.Context, caller CallerScope, filter report_dto.ExportFilter) (report_dto.ExportResult, error) {
	form, err := s.formRepo.FindFormByCode(ctx, filter.FormCode)
	if err != nil {
		return report_dto.ExportResult{}, apperror.NotFound("Form tidak ditemukan")
	}
	var orgIDs []int64
	if caller.RequestedOrganizationID != nil {
		ok, err := s.orgScope.IsAccessible(ctx, caller.OrganizationID, caller.OrganizationTypeCode, caller.WildcardTierAccess, *caller.RequestedOrganizationID)
		if err != nil {
			return report_dto.ExportResult{}, apperror.Internal("")
		}
		if !ok {
			return report_dto.ExportResult{}, apperror.Forbidden("Anda tidak memiliki akses ke organisasi ini")
		}
		orgIDs, err = s.orgScope.AccessibleOrganizationIDsForTarget(ctx, *caller.RequestedOrganizationID)
		if err != nil {
			return report_dto.ExportResult{}, apperror.Internal("")
		}
	} else {
		var err error
		orgIDs, err = s.orgScope.AccessibleOrganizationIDs(ctx, caller.OrganizationID, caller.OrganizationTypeCode, caller.WildcardTierAccess)
		if err != nil {
			return report_dto.ExportResult{}, apperror.Internal("")
		}
	}
	rows, err := s.repo.SubmissionRows(ctx, form.FormID, filter.Status, orgIDs)
	if err != nil {
		return report_dto.ExportResult{}, apperror.Internal("")
	}

	var data []byte
	var contentType, ext string
	switch filter.Format {
	case "csv":
		data, err = toCSV(rows)
		contentType, ext = "text/csv", "csv"
	default:
		data, err = toXLSX(rows)
		contentType, ext = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "xlsx"
	}
	if err != nil {
		return report_dto.ExportResult{}, apperror.Internal("Gagal membuat berkas laporan")
	}

	fileName := fmt.Sprintf("laporan-%s-%s.%s", filter.FormCode, time.Now().Format("20060102-150405"), ext)
	var actorOrg *int64
	if caller.OrganizationID != nil {
		actorOrg = caller.OrganizationID
	}
	s.audit.LogExport(ctx, auditlog.Entry{
		ActorUserID: caller.UserID, ActorOrganizationID: actorOrg,
		Action: "EXPORT", Entity: "tr_submission", EntityID: form.FormID,
		Metadata: map[string]interface{}{"formCode": filter.FormCode, "status": filter.Status, "format": filter.Format, "rowCount": len(rows)},
	})

	return report_dto.ExportResult{FileName: fileName, ContentType: contentType, Data: data}, nil
}

// ---------- Kantong Amal (Phase 9) — §15 techspec ----------

func (s *ServiceImpl) GetBalanceReport(ctx context.Context, f report_dto.BalanceReportFilter) (report_dto.BalanceReportResponse, error) {
	opening, err := s.repo.BalanceBefore(ctx, f.CampaignID, f.From)
	if err != nil {
		return report_dto.BalanceReportResponse{}, apperror.Internal("")
	}
	incoming, err := s.repo.LedgerSumByType(ctx, f.CampaignID, constants.LedgerEntryDonationCredit, f.From, f.To)
	if err != nil {
		return report_dto.BalanceReportResponse{}, apperror.Internal("")
	}
	outgoing, err := s.repo.WithdrawalSuccessSum(ctx, f.CampaignID, f.From, f.To)
	if err != nil {
		return report_dto.BalanceReportResponse{}, apperror.Internal("")
	}
	refund, err := s.repo.LedgerSumByType(ctx, f.CampaignID, constants.LedgerEntryRefundDebit, f.From, f.To)
	if err != nil {
		return report_dto.BalanceReportResponse{}, apperror.Internal("")
	}
	adjCredit, err := s.repo.LedgerSumByType(ctx, f.CampaignID, constants.LedgerEntryAdjustmentCredit, f.From, f.To)
	if err != nil {
		return report_dto.BalanceReportResponse{}, apperror.Internal("")
	}
	adjDebit, err := s.repo.LedgerSumByType(ctx, f.CampaignID, constants.LedgerEntryAdjustmentDebit, f.From, f.To)
	if err != nil {
		return report_dto.BalanceReportResponse{}, apperror.Internal("")
	}
	fee, err := s.repo.FeeSum(ctx, f.CampaignID, f.From, f.To)
	if err != nil {
		return report_dto.BalanceReportResponse{}, apperror.Internal("")
	}
	closing, err := s.repo.BalanceAsOf(ctx, f.CampaignID, f.To)
	if err != nil {
		return report_dto.BalanceReportResponse{}, apperror.Internal("")
	}

	adjustment := adjCredit - adjDebit
	expectedClosing := opening + incoming - outgoing - refund + adjustment

	return report_dto.BalanceReportResponse{
		From: f.From, To: f.To, CampaignID: f.CampaignID,
		OpeningBalance: opening, Incoming: incoming, Outgoing: outgoing, Refund: refund,
		Adjustment: adjustment, Fee: fee, ClosingBalance: closing, ExpectedClosing: expectedClosing,
		IsBalanced: math.Abs(closing-expectedClosing) < 1,
	}, nil
}

func balanceReportColumns() []string {
	return []string{"Dari", "Sampai", "Saldo Awal", "Masuk (Donasi)", "Keluar (Withdrawal)", "Refund", "Adjustment", "Fee", "Saldo Akhir", "Saldo Akhir (Ekspektasi)", "Seimbang"}
}

func balanceReportValues(r report_dto.BalanceReportResponse) []string {
	yesNo := "Ya"
	if !r.IsBalanced {
		yesNo = "TIDAK — perlu ditinjau"
	}
	return []string{
		r.From.Format("2006-01-02"), r.To.Format("2006-01-02"),
		formatMoney(r.OpeningBalance), formatMoney(r.Incoming), formatMoney(r.Outgoing), formatMoney(r.Refund),
		formatMoney(r.Adjustment), formatMoney(r.Fee), formatMoney(r.ClosingBalance), formatMoney(r.ExpectedClosing), yesNo,
	}
}

func (s *ServiceImpl) ExportBalanceReport(ctx context.Context, actorUserID int64, f report_dto.BalanceReportFilter) (report_dto.ExportResult, error) {
	report, err := s.GetBalanceReport(ctx, f)
	if err != nil {
		return report_dto.ExportResult{}, err
	}
	data, err := toXLSXGeneric("Laporan Saldo", balanceReportColumns(), [][]string{balanceReportValues(report)})
	if err != nil {
		return report_dto.ExportResult{}, apperror.Internal("Gagal membuat berkas laporan")
	}
	fileName := fmt.Sprintf("laporan-saldo-%s.xlsx", time.Now().Format("20060102-150405"))
	s.audit.LogExport(ctx, auditlog.Entry{
		ActorUserID: actorUserID, Action: "EXPORT", Entity: "balance_report", EntityID: f.CampaignID,
		Metadata: map[string]interface{}{"from": f.From, "to": f.To},
	})
	return report_dto.ExportResult{FileName: fileName, ContentType: xlsxContentType, Data: data}, nil
}

func (s *ServiceImpl) ListCampaignReport(ctx context.Context, f report_dto.KantongAmalReportFilter) ([]report_dto.CampaignReportRow, int, error) {
	rows, total, err := s.repo.CampaignReportRows(ctx, f)
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	return rows, int(total), nil
}

func campaignReportColumns() []string {
	return []string{"Campaign", "Status", "Target", "Terkumpul", "% Tercapai", "Donor Unik", "Jumlah Transaksi", "Mulai", "Selesai"}
}

func campaignReportValues(r report_dto.CampaignReportRow) []string {
	pct := "0%"
	if r.TargetAmount > 0 {
		pct = fmt.Sprintf("%.1f%%", r.CollectedAmount/r.TargetAmount*100)
	}
	return []string{
		r.Title, r.Status, formatMoney(r.TargetAmount), formatMoney(r.CollectedAmount), pct,
		fmt.Sprintf("%d", r.DonorCount), fmt.Sprintf("%d", r.TransactionCount),
		formatNullDate(r.StartDate), formatNullDate(r.EndDate),
	}
}

func (s *ServiceImpl) ExportCampaignReport(ctx context.Context, actorUserID int64, f report_dto.KantongAmalReportFilter) (report_dto.ExportResult, error) {
	f.Page, f.Limit = 1, 10000 // export tidak dipaginasi — batas atas wajar, bukan streaming (lihat Notes phase summary)
	rows, _, err := s.repo.CampaignReportRows(ctx, f)
	if err != nil {
		return report_dto.ExportResult{}, apperror.Internal("")
	}
	values := make([][]string, 0, len(rows))
	for _, r := range rows {
		values = append(values, campaignReportValues(r))
	}
	data, err := toXLSXGeneric("Laporan Campaign", campaignReportColumns(), values)
	if err != nil {
		return report_dto.ExportResult{}, apperror.Internal("Gagal membuat berkas laporan")
	}
	fileName := fmt.Sprintf("laporan-campaign-%s.xlsx", time.Now().Format("20060102-150405"))
	s.audit.LogExport(ctx, auditlog.Entry{
		ActorUserID: actorUserID, Action: "EXPORT", Entity: "campaign_report", EntityID: f.CampaignID,
		Metadata: map[string]interface{}{"rowCount": len(rows)},
	})
	return report_dto.ExportResult{FileName: fileName, ContentType: xlsxContentType, Data: data}, nil
}

func (s *ServiceImpl) ListDonationReport(ctx context.Context, f report_dto.KantongAmalReportFilter) ([]report_dto.DonationReportRow, int, error) {
	rows, total, err := s.repo.DonationReportRows(ctx, f)
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	return rows, int(total), nil
}

func donationReportColumns() []string {
	return []string{"Campaign", "Donatur", "Anonim", "Nominal", "Fee", "Total Dibayar", "Status", "Gateway", "Tanggal"}
}

func donationReportValues(r report_dto.DonationReportRow) []string {
	anon := "Tidak"
	if r.IsAnonymous {
		anon = "Ya"
	}
	return []string{
		r.CampaignTitle, r.DonorName, anon, formatMoney(r.Amount), formatMoney(r.AdminFee), formatMoney(r.TotalAmount),
		r.PaymentStatus, r.Gateway, r.CreatedDate.Format("2006-01-02 15:04"),
	}
}

func (s *ServiceImpl) ExportDonationReport(ctx context.Context, actorUserID int64, f report_dto.KantongAmalReportFilter) (report_dto.ExportResult, error) {
	f.Page, f.Limit = 1, 10000
	rows, _, err := s.repo.DonationReportRows(ctx, f)
	if err != nil {
		return report_dto.ExportResult{}, apperror.Internal("")
	}
	values := make([][]string, 0, len(rows))
	for _, r := range rows {
		values = append(values, donationReportValues(r))
	}
	data, err := toXLSXGeneric("Laporan Donasi", donationReportColumns(), values)
	if err != nil {
		return report_dto.ExportResult{}, apperror.Internal("Gagal membuat berkas laporan")
	}
	fileName := fmt.Sprintf("laporan-donasi-%s.xlsx", time.Now().Format("20060102-150405"))
	s.audit.LogExport(ctx, auditlog.Entry{
		ActorUserID: actorUserID, Action: "EXPORT", Entity: "donation_report", EntityID: f.CampaignID,
		Metadata: map[string]interface{}{"rowCount": len(rows)},
	})
	return report_dto.ExportResult{FileName: fileName, ContentType: xlsxContentType, Data: data}, nil
}

func (s *ServiceImpl) ListWithdrawalReport(ctx context.Context, f report_dto.KantongAmalReportFilter) ([]report_dto.WithdrawalReportRow, int, []report_dto.WithdrawalStatusFunnel, error) {
	rows, total, err := s.repo.WithdrawalReportRows(ctx, f)
	if err != nil {
		return nil, 0, nil, apperror.Internal("")
	}
	funnel, err := s.repo.WithdrawalStatusFunnel(ctx, f.CampaignID)
	if err != nil {
		return nil, 0, nil, apperror.Internal("")
	}
	return rows, int(total), funnel, nil
}

func withdrawalReportColumns() []string {
	return []string{"Ref", "Campaign", "Nominal", "Fee", "Net", "Status", "Bank", "No. Rekening", "Diajukan", "Disetujui", "Diproses", "Selesai"}
}

func withdrawalReportValues(r report_dto.WithdrawalReportRow) []string {
	return []string{
		r.WithdrawalRef, r.CampaignTitle, formatMoney(r.Amount), formatMoney(r.Fee), formatMoney(r.NetAmount), r.Status,
		r.BeneficiaryBankCode, r.BeneficiaryAccountNumber, r.RequestedDate.Format("2006-01-02 15:04"),
		formatNullDateTime(r.ApprovedDate), formatNullDateTime(r.ProcessingDate), formatNullDateTime(r.CompletedDate),
	}
}

func (s *ServiceImpl) ExportWithdrawalReport(ctx context.Context, actorUserID int64, f report_dto.KantongAmalReportFilter) (report_dto.ExportResult, error) {
	f.Page, f.Limit = 1, 10000
	rows, _, err := s.repo.WithdrawalReportRows(ctx, f)
	if err != nil {
		return report_dto.ExportResult{}, apperror.Internal("")
	}
	values := make([][]string, 0, len(rows))
	for _, r := range rows {
		values = append(values, withdrawalReportValues(r))
	}
	data, err := toXLSXGeneric("Laporan Withdrawal", withdrawalReportColumns(), values)
	if err != nil {
		return report_dto.ExportResult{}, apperror.Internal("Gagal membuat berkas laporan")
	}
	fileName := fmt.Sprintf("laporan-withdrawal-%s.xlsx", time.Now().Format("20060102-150405"))
	s.audit.LogExport(ctx, auditlog.Entry{
		ActorUserID: actorUserID, Action: "EXPORT", Entity: "withdrawal_report", EntityID: f.CampaignID,
		Metadata: map[string]interface{}{"rowCount": len(rows)},
	})
	return report_dto.ExportResult{FileName: fileName, ContentType: xlsxContentType, Data: data}, nil
}

// RunReconciliation membandingkan ledger internal terhadap saldo wallet
// gateway sungguhan (§15.1/§15.5) dan menyimpannya sebagai snapshot baru.
// Perbandingan transaksi gateway satu-per-satu (ListTransactions/
// DetailTransaction tiap donasi) TIDAK diimplementasikan di sini — di luar
// scope Phase 9, dicatat di Notes phase summary; yang diimplementasikan
// adalah rekonsiliasi level-wallet konkret dengan threshold eksplisit §15.1.
func (s *ServiceImpl) RunReconciliation(ctx context.Context) (report_dto.ReconciliationSnapshotResponse, error) {
	now := time.Now()

	donationCount, donationAmount, err := s.repo.DonationPaidTotals(ctx)
	if err != nil {
		return report_dto.ReconciliationSnapshotResponse{}, apperror.Internal("")
	}
	ledgerDonationCredit, err := s.repo.LedgerTotalByType(ctx, constants.LedgerEntryDonationCredit)
	if err != nil {
		return report_dto.ReconciliationSnapshotResponse{}, apperror.Internal("")
	}
	withdrawalCount, withdrawalAmount, err := s.repo.WithdrawalSuccessTotals(ctx)
	if err != nil {
		return report_dto.ReconciliationSnapshotResponse{}, apperror.Internal("")
	}
	expectedBalance, err := s.repo.BalanceAsOf(ctx, 0, now)
	if err != nil {
		return report_dto.ReconciliationSnapshotResponse{}, apperror.Internal("")
	}

	// recentlyPaid = donasi PAID dalam settlementWindow menit terakhir — belum
	// tentu sudah settle penuh di wallet gateway saat snapshot ini diambil.
	// Dihitung selalu (bukan cuma saat gateway sukses) supaya nilainya tetap
	// tersimpan & bisa ditampilkan di histori walau gatewayBalance gagal diambil.
	settlementMinutes := s.cfg.BisatopupSettlementMinutesCrowdfunding
	settlementWindow := time.Duration(settlementMinutes) * time.Minute
	recentlyPaid, _ := s.repo.LedgerSumByType(ctx, 0, constants.LedgerEntryDonationCredit, now.Add(-settlementWindow), now)

	var gatewayBalance float64
	var gatewayErrMsg string
	hasAnomaly := false
	walletRes, gerr := s.gateway.WalletBalance(ctx)
	if gerr != nil {
		gatewayErrMsg = gerr.Error()
		log.Printf("[REPORT] reconciliation: gagal ambil wallet balance gateway: %v", gerr)
	} else {
		gatewayBalance = float64(walletRes.Amount)
	}
	discrepancy := gatewayBalance - expectedBalance
	if gerr == nil {
		// Toleransi tambahan untuk donasi yang baru PAID dan belum settle
		// penuh di wallet gateway (§15.1 — atribusi "Settling..."), setara
		// "Settlement Pending" di ldksyahid-app (balance-report.blade.php).
		allowedGap := reconciliationDiscrepancyThreshold + recentlyPaid
		hasAnomaly = math.Abs(discrepancy) > allowedGap
	}

	params := report_dto.ReconciliationSnapshotParams{
		SnapshotDate: now, DonationPaidCount: donationCount, DonationPaidAmount: donationAmount,
		LedgerDonationCreditAmount: ledgerDonationCredit, WithdrawalSuccessCount: withdrawalCount,
		WithdrawalSuccessAmount: withdrawalAmount, ExpectedBalance: expectedBalance,
		GatewayWalletBalance: gatewayBalance, DiscrepancyAmount: discrepancy,
		SettlementPendingAmount: recentlyPaid, SettlementMinutes: settlementMinutes,
		HasAnomaly: hasAnomaly, GatewayError: gatewayErrMsg,
	}
	id, err := s.repo.CreateReconciliationSnapshot(ctx, params)
	if err != nil {
		return report_dto.ReconciliationSnapshotResponse{}, apperror.Internal("")
	}
	if hasAnomaly {
		log.Printf("[REPORT] reconciliation: ANOMALI terdeteksi, discrepancy=%.2f (ambang %.2f), snapshotID=%d", discrepancy, float64(reconciliationDiscrepancyThreshold), id)
	}

	return report_dto.ReconciliationSnapshotResponse{
		SnapshotID: id, SnapshotDate: params.SnapshotDate,
		DonationPaidCount: params.DonationPaidCount, DonationPaidAmount: params.DonationPaidAmount,
		LedgerDonationCreditAmount: params.LedgerDonationCreditAmount,
		WithdrawalSuccessCount:     params.WithdrawalSuccessCount, WithdrawalSuccessAmount: params.WithdrawalSuccessAmount,
		ExpectedBalance: params.ExpectedBalance, GatewayWalletBalance: params.GatewayWalletBalance,
		DiscrepancyAmount: params.DiscrepancyAmount,
		SettlementPendingAmount: params.SettlementPendingAmount, SettlementMinutes: params.SettlementMinutes,
		HasAnomaly: params.HasAnomaly, CreatedDate: now,
	}, nil
}

func (s *ServiceImpl) ListReconciliationHistory(ctx context.Context, q dto.ListQuery) ([]report_dto.ReconciliationSnapshotResponse, int, error) {
	rows, total, err := s.repo.ListReconciliationSnapshots(ctx, q)
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	return rows, int(total), nil
}

func (s *ServiceImpl) ListGlobalLedger(ctx context.Context, f report_dto.GlobalLedgerFilter) ([]report_dto.GlobalLedgerRow, int, error) {
	rows, total, err := s.repo.GlobalLedgerRows(ctx, f)
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	return rows, int(total), nil
}

func (s *ServiceImpl) GetAnalytics(ctx context.Context, campaignID int64) (report_dto.AnalyticsResponse, error) {
	amountBands, err := s.repo.DonationAmountBands(ctx, campaignID)
	if err != nil {
		return report_dto.AnalyticsResponse{}, apperror.Internal("")
	}
	ageBands, err := s.repo.DonorAgeBands(ctx, campaignID)
	if err != nil {
		return report_dto.AnalyticsResponse{}, apperror.Internal("")
	}
	progress, _, err := s.repo.CampaignReportRows(ctx, report_dto.KantongAmalReportFilter{CampaignID: campaignID, Page: 1, Limit: 100})
	if err != nil {
		return report_dto.AnalyticsResponse{}, apperror.Internal("")
	}
	return report_dto.AnalyticsResponse{DonationAmountBands: amountBands, DonorAgeBands: ageBands, CampaignProgress: progress}, nil
}

func (s *ServiceImpl) ListFinanceAuditLog(ctx context.Context, f report_dto.FinanceAuditLogFilter) ([]report_dto.FinanceAuditLogItem, int, error) {
	rows, total, err := s.repo.ListFinanceAuditLog(ctx, f)
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	return rows, int(total), nil
}

// RunReconciliationScheduler menjalankan RunReconciliation tiap
// reconciliationCheckInterval sampai proses berhenti — pola sama
// donation_service.RunExpireScheduler/withdrawal_service.RunReconcileScheduler.
func (s *ServiceImpl) RunReconciliationScheduler() {
	ticker := time.NewTicker(reconciliationCheckInterval)
	defer ticker.Stop()
	for range ticker.C {
		if _, err := s.RunReconciliation(context.Background()); err != nil {
			log.Printf("[REPORT] daily_reconciliation: gagal jalan: %v", err)
		}
	}
}

const xlsxContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// toXLSXGeneric membangun satu berkas XLSX in-memory (bukan streamed —
// lihat Notes phase summary untuk alasan menyimpang dari saran
// excelize.StreamWriter §15.6) dari header + baris string apa pun.
func toXLSXGeneric(sheet string, columns []string, rows [][]string) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	f.SetSheetName("Sheet1", sheet)
	for i, col := range columns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, col)
	}
	for r, row := range rows {
		for c, v := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			f.SetCellValue(sheet, cell, v)
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func formatMoney(v float64) string {
	return fmt.Sprintf("%.2f", v)
}

func formatNullDate(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format("2006-01-02")
}

func formatNullDateTime(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format("2006-01-02 15:04")
}
