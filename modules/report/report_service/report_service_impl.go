package report_service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"fsldk-api/base/apperror"
	"fsldk-api/config"
	"fsldk-api/constants"
	"fsldk-api/modules/report/report_dto"
	"fsldk-api/modules/report/report_repository"
	"fsldk-api/modules/submission_form/submission_form_model"
	"fsldk-api/modules/submission_form/submission_form_repository"
	"fsldk-api/pkg/auditlog"
	"fsldk-api/pkg/bisatopup"

	"github.com/xuri/excelize/v2"
)

// reconciliationDiscrepancyThreshold adalah ambang selisih ledger vs wallet
// gateway yang dianggap wajar sebelum di-flag anomali — keputusan final
// OQ-22 (reuse ldksyahid-app): Rp50.000.
const reconciliationDiscrepancyThreshold = 50_000

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

// answerSheetBaseColumns mendahului kolom per-field di tiap sheet "Jawaban" —
// konteks organisasi yang sama dipakai di sheet "Ringkasan", diulang di sini
// supaya tiap sheet jawaban tetap bisa dibaca berdiri sendiri.
var answerSheetBaseColumns = []string{"Nama Organisasi", "Provinsi", "Kota/Kabupaten", "Status", "Tanggal Submit"}

func rowsToValues(rows []report_dto.SubmissionRow) [][]string {
	out := make([][]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowValues(row))
	}
	return out
}

// sheetData adalah satu sheet mentah (nama + header + baris) sebelum ditulis
// ke berkas XLSX — dipisah dari excelize supaya ExportSubmissions bisa
// menyusun N sheet (Ringkasan + satu per versi form) sebelum menulis sekali.
type sheetData struct {
	Name    string
	Columns []string
	Rows    [][]string
}

// buildWorkbook menulis satu berkas XLSX in-memory berisi beberapa sheet
// sekaligus — generalisasi toXLSXGeneric (yang cuma satu sheet) untuk
// ExportSubmissions, satu-satunya pemanggil yang butuh banyak sheet saat ini
// (Ringkasan + Jawaban per versi form, lihat buildAnswerSheets).
func buildWorkbook(sheets []sheetData) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	usedNames := map[string]bool{}
	for i, sd := range sheets {
		name := uniqueSheetName(sd.Name, usedNames)
		if i == 0 {
			f.SetSheetName("Sheet1", name)
		} else if _, err := f.NewSheet(name); err != nil {
			return nil, err
		}
		for c, col := range sd.Columns {
			cell, _ := excelize.CoordinatesToCellName(c+1, 1)
			f.SetCellValue(name, cell, col)
		}
		for r, row := range sd.Rows {
			for c, v := range row {
				cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
				f.SetCellValue(name, cell, v)
			}
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// uniqueSheetName menegakkan batasan nama sheet Excel (maks 31 karakter,
// tanpa \/?*[]:, tidak boleh duplikat) — dipanggil berurutan per sheet,
// `used` diisi in-place supaya sheet berikutnya tahu nama yang sudah dipakai.
func uniqueSheetName(raw string, used map[string]bool) string {
	replacer := strings.NewReplacer(`\`, "-", "/", "-", "?", "", "*", "", "[", "(", "]", ")", ":", "-")
	name := strings.TrimSpace(replacer.Replace(raw))
	if name == "" {
		name = "Sheet"
	}
	if len(name) > 31 {
		name = name[:31]
	}
	base, suffix := name, 2
	for used[strings.ToLower(name)] {
		extra := fmt.Sprintf(" (%d)", suffix)
		maxBase := 31 - len(extra)
		if maxBase < 1 {
			maxBase = 1
		}
		trimmed := base
		if len(trimmed) > maxBase {
			trimmed = trimmed[:maxBase]
		}
		name = trimmed + extra
		suffix++
	}
	used[strings.ToLower(name)] = true
	return name
}

// buildAnswerSheets mengelompokkan submission berdasarkan formVersionID lalu
// menyusun satu sheet "Jawaban" per versi — field Levelisasi LDK bisa berubah
// tiap Puskomnas publish versi baru (lihat Publish Cascade), jadi kolomnya
// TIDAK bisa digabung lintas versi tanpa campur aduk. Versi terbaru (ID
// tertinggi) ditaruh duluan.
func (s *ServiceImpl) buildAnswerSheets(ctx context.Context, rows []report_dto.SubmissionRow) ([]sheetData, error) {
	byVersion := map[int64][]report_dto.SubmissionRow{}
	var versionIDs []int64
	for _, r := range rows {
		if _, ok := byVersion[r.FormVersionID]; !ok {
			versionIDs = append(versionIDs, r.FormVersionID)
		}
		byVersion[r.FormVersionID] = append(byVersion[r.FormVersionID], r)
	}
	sort.Slice(versionIDs, func(i, j int) bool { return versionIDs[i] > versionIDs[j] })

	sheets := make([]sheetData, 0, len(versionIDs))
	for _, versionID := range versionIDs {
		group := byVersion[versionID]
		version, err := s.formRepo.FindVersionByID(ctx, versionID)
		if err != nil {
			return nil, err
		}
		sections, err := s.formRepo.ListSectionsByVersion(ctx, versionID)
		if err != nil {
			return nil, err
		}
		fields, err := s.formRepo.ListFieldsByVersion(ctx, versionID)
		if err != nil {
			return nil, err
		}
		options, err := s.formRepo.ListOptionsByVersion(ctx, versionID)
		if err != nil {
			return nil, err
		}

		sectionLabel := make(map[int64]string, len(sections))
		sectionOrder := make(map[int64]int, len(sections))
		for _, sec := range sections {
			sectionLabel[sec.SectionID] = sec.SectionLabel
			sectionOrder[sec.SectionID] = sec.SortOrder
		}
		sort.Slice(fields, func(i, j int) bool {
			oi, oj := sectionOrder[fields[i].SectionID], sectionOrder[fields[j].SectionID]
			if oi != oj {
				return oi < oj
			}
			return fields[i].SortOrder < fields[j].SortOrder
		})

		optionsByField := make(map[int64][]submission_form_model.Option)
		optionValueByID := make(map[int64]string, len(options))
		for _, o := range options {
			optionsByField[o.FieldID] = append(optionsByField[o.FieldID], o)
			optionValueByID[o.OptionID] = o.OptionValue
		}
		fieldByID := make(map[int64]submission_form_model.Field, len(fields))
		for _, f := range fields {
			fieldByID[f.FieldID] = f
		}

		submissionIDs := make([]int64, len(group))
		for i, r := range group {
			submissionIDs[i] = r.SubmissionID
		}
		answers, err := s.repo.AnswersBySubmissionIDs(ctx, submissionIDs)
		if err != nil {
			return nil, err
		}
		answersBySubmission := make(map[int64]map[int64]report_dto.AnswerRow, len(group))
		for _, a := range answers {
			bucket, ok := answersBySubmission[a.SubmissionID]
			if !ok {
				bucket = make(map[int64]report_dto.AnswerRow)
				answersBySubmission[a.SubmissionID] = bucket
			}
			bucket[a.FieldID] = a
		}

		columns := append([]string{}, answerSheetBaseColumns...)
		for _, f := range fields {
			label := f.FieldLabel
			if sl := sectionLabel[f.SectionID]; sl != "" {
				label = sl + " — " + f.FieldLabel
			}
			columns = append(columns, label)
		}

		rowsOut := make([][]string, 0, len(group))
		for _, subRow := range group {
			answersForSub := answersBySubmission[subRow.SubmissionID]
			values := make([]string, 0, len(columns))
			values = append(values,
				subRow.OrganizationName, subRow.ProvinceName.String, subRow.CityName.String,
				subRow.Status, formatNullDateTime(subRow.SubmittedDate),
			)
			for _, f := range fields {
				if !isFieldVisible(f, fieldByID, answersForSub, optionValueByID) {
					values = append(values, "")
					continue
				}
				values = append(values, formatAnswerValue(f, answersForSub[f.FieldID], optionsByField[f.FieldID]))
			}
			rowsOut = append(rowsOut, values)
		}

		sheets = append(sheets, sheetData{
			Name:    fmt.Sprintf("Jawaban V%d", version.VersionNumber),
			Columns: columns,
			Rows:    rowsOut,
		})
	}
	return sheets, nil
}

// answerConditionalRule mencerminkan bentuk JSON conditionalRuleJSON yang
// sama persis dipakai submission_service (evaluateConditional) dan frontend
// SubmissionAnswersViewComponent — duplikasi kecil yang disengaja (report
// module tidak bergantung ke submission_service, cuma submission_form_repository).
type answerConditionalRule struct {
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

// isFieldVisible meniru persis evaluateConditional (submission_service) +
// SubmissionAnswersViewComponent.isVisible (frontend) — field tanpa aturan
// selalu tampil; field kondisional disembunyikan (nilainya dikosongkan di
// ekspor, BUKAN dihapus dari tr_submission_answer) kalau field acuannya belum
// dijawab atau jawabannya tidak memenuhi aturan.
func isFieldVisible(f submission_form_model.Field, byID map[int64]submission_form_model.Field, answers map[int64]report_dto.AnswerRow, optionValueByID map[int64]string) bool {
	if !f.ConditionalOnFieldID.Valid || !f.ConditionalRuleJSON.Valid || f.ConditionalRuleJSON.String == "" {
		return true
	}
	trigger, ok := byID[f.ConditionalOnFieldID.Int64]
	if !ok {
		return false
	}
	var rule answerConditionalRule
	if err := json.Unmarshal([]byte(f.ConditionalRuleJSON.String), &rule); err != nil {
		return true
	}
	triggerAnswer, hasAnswer := answers[trigger.FieldID]
	if !hasAnswer {
		return false
	}
	actual := answerAsString(triggerAnswer, optionValueByID)
	switch rule.Operator {
	case "equals":
		return actual == rule.Value
	case "notEquals":
		return actual != rule.Value
	default:
		return true
	}
}

// answerAsString mengembalikan representasi string MENTAH satu jawaban
// (optionVALUE, bukan label, untuk SELECT/RADIO) — dipakai evaluasi
// conditional di atas, mencerminkan submission_service.answerAsString.
func answerAsString(a report_dto.AnswerRow, optionValueByID map[int64]string) string {
	switch {
	case a.ValueOptionID.Valid:
		return optionValueByID[a.ValueOptionID.Int64]
	case a.ValueText.Valid:
		return a.ValueText.String
	case a.ValueNumber.Valid:
		return strconv.FormatFloat(a.ValueNumber.Float64, 'f', -1, 64)
	case a.ValueDate.Valid:
		return a.ValueDate.Time.Format("2006-01-02")
	default:
		return ""
	}
}

// formatAnswerValue mengembalikan nilai TAMPILAN (label, bukan value mentah)
// satu jawaban sesuai fieldType — mencerminkan
// SubmissionAnswersViewComponent.displayValue di frontend. MULTISELECT/
// CHECKBOX disimpan sebagai JSON array optionID di valueText (lihat
// submission_service_impl.go buildAnswerParams), di-decode lalu
// dipetakan ke optionLabel gabungan koma.
func formatAnswerValue(f submission_form_model.Field, a report_dto.AnswerRow, options []submission_form_model.Option) string {
	optionLabel := func(id int64) string {
		for _, o := range options {
			if o.OptionID == id {
				return o.OptionLabel
			}
		}
		return ""
	}
	switch f.FieldType {
	case constants.FieldTypeSelect, constants.FieldTypeRadio:
		if !a.ValueOptionID.Valid {
			return ""
		}
		return optionLabel(a.ValueOptionID.Int64)
	case constants.FieldTypeMultiselect, constants.FieldTypeCheckbox:
		if !a.ValueText.Valid || a.ValueText.String == "" {
			return ""
		}
		var ids []int64
		if err := json.Unmarshal([]byte(a.ValueText.String), &ids); err != nil {
			return ""
		}
		labels := make([]string, 0, len(ids))
		for _, id := range ids {
			if l := optionLabel(id); l != "" {
				labels = append(labels, l)
			}
		}
		return strings.Join(labels, ", ")
	case constants.FieldTypeFileDoc, constants.FieldTypeFileImage:
		return a.ValueFileName.String
	case constants.FieldTypeNumber:
		if !a.ValueNumber.Valid {
			return ""
		}
		return strconv.FormatFloat(a.ValueNumber.Float64, 'f', -1, 64)
	case constants.FieldTypeDate:
		if !a.ValueDate.Valid {
			return ""
		}
		return a.ValueDate.Time.Format("2006-01-02")
	default:
		return a.ValueText.String
	}
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
		// CSV cuma satu tabel datar — tidak ada konsep sheet, jadi tetap
		// ringkasan saja (tanpa jawaban per field). Ekspor lengkap dengan
		// jawaban HARUS pakai xlsx.
		data, err = toCSV(rows)
		contentType, ext = "text/csv", "csv"
	default:
		sheets := []sheetData{{Name: "Ringkasan", Columns: reportColumns, Rows: rowsToValues(rows)}}
		answerSheets, aerr := s.buildAnswerSheets(ctx, rows)
		if aerr != nil {
			return report_dto.ExportResult{}, apperror.Internal("Gagal menyusun data jawaban")
		}
		sheets = append(sheets, answerSheets...)
		data, err = buildWorkbook(sheets)
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

func (s *ServiceImpl) GetWithdrawalStatusFunnel(ctx context.Context, campaignID int64) ([]report_dto.WithdrawalStatusFunnel, error) {
	funnel, err := s.repo.WithdrawalStatusFunnel(ctx, campaignID)
	if err != nil {
		return nil, apperror.Internal("")
	}
	return funnel, nil
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

// GetReconciliation membandingkan ledger internal terhadap saldo wallet
// gateway sungguhan (§15.1/§15.5) secara LIVE — dihitung ulang setiap
// dipanggil, tidak lagi disimpan sebagai snapshot histori (Balance Report
// real-time, revision-prompt-4.md). Perbandingan transaksi gateway
// satu-per-satu (ListTransactions/DetailTransaction tiap donasi) TIDAK
// diimplementasikan di sini — di luar scope Phase 9, dicatat di Notes phase
// summary; yang diimplementasikan adalah rekonsiliasi level-wallet konkret
// dengan threshold eksplisit §15.1.
func (s *ServiceImpl) GetReconciliation(ctx context.Context) (report_dto.ReconciliationResponse, error) {
	now := time.Now()

	donationCount, donationAmount, err := s.repo.DonationPaidTotals(ctx)
	if err != nil {
		return report_dto.ReconciliationResponse{}, apperror.Internal("")
	}
	ledgerDonationCredit, err := s.repo.LedgerTotalByType(ctx, constants.LedgerEntryDonationCredit)
	if err != nil {
		return report_dto.ReconciliationResponse{}, apperror.Internal("")
	}
	withdrawalCount, withdrawalAmount, err := s.repo.WithdrawalSuccessTotals(ctx)
	if err != nil {
		return report_dto.ReconciliationResponse{}, apperror.Internal("")
	}
	expectedBalance, err := s.repo.BalanceAsOf(ctx, 0, now)
	if err != nil {
		return report_dto.ReconciliationResponse{}, apperror.Internal("")
	}

	// recentlyPaid = donasi PAID dalam settlementWindow menit terakhir — belum
	// tentu sudah settle penuh di wallet gateway saat pemeriksaan ini
	// dijalankan. Dihitung selalu (bukan cuma saat gateway sukses) supaya
	// nilainya tetap bisa ditampilkan walau gatewayBalance gagal diambil.
	settlementMinutes := s.cfg.BisatopupSettlementMinutesCrowdfunding
	settlementWindow := time.Duration(settlementMinutes) * time.Minute
	recentlyPaid, _ := s.repo.LedgerSumByType(ctx, 0, constants.LedgerEntryDonationCredit, now.Add(-settlementWindow), now)

	var gatewayBalance float64
	var gatewayErrMsg *string
	hasAnomaly := false
	walletRes, gerr := s.gateway.WalletBalance(ctx)
	if gerr != nil {
		msg := gerr.Error()
		gatewayErrMsg = &msg
		log.Printf("[REPORT] reconciliation: gagal ambil wallet balance gateway: %v", gerr)
	} else {
		gatewayBalance = float64(walletRes.Amount)
	}
	discrepancy := gatewayBalance - expectedBalance
	// settlementPending = bagian dari recentlyPaid yang KEMUNGKINAN masih
	// benar-benar outstanding di gateway — dibatasi maksimum sebesar
	// discrepancy aktual saat ini, bukan seluruh donasi yang baru PAID dalam
	// jendela waktu. Tanpa pembatasan ini, badge "Settlement Pending" tetap
	// menampilkan angka penuh meski discrepancy sudah 0 (settlement selesai
	// duluan sebelum jendela waktu habis), padahal seharusnya otomatis
	// hilang begitu uangnya benar-benar sudah sampai di wallet gateway.
	var settlementPending float64
	if gerr == nil {
		// Toleransi tambahan untuk donasi yang baru PAID dan belum settle
		// penuh di wallet gateway (§15.1 — atribusi "Settling..."), setara
		// "Settlement Pending" di ldksyahid-app (balance-report.blade.php).
		allowedGap := reconciliationDiscrepancyThreshold + recentlyPaid
		hasAnomaly = math.Abs(discrepancy) > allowedGap
		// Hanya relevan saat gateway MASIH kekurangan dari ekspektasi
		// (discrepancy negatif) — kalau gateway sudah sama/lebih besar,
		// tidak ada apa pun yang masih pending.
		if discrepancy < 0 {
			settlementPending = math.Min(recentlyPaid, -discrepancy)
		}
	}
	if hasAnomaly {
		log.Printf("[REPORT] reconciliation: ANOMALI terdeteksi, discrepancy=%.2f (ambang %.2f)", discrepancy, float64(reconciliationDiscrepancyThreshold))
	}

	return report_dto.ReconciliationResponse{
		DonationPaidCount: donationCount, DonationPaidAmount: donationAmount,
		LedgerDonationCreditAmount: ledgerDonationCredit,
		WithdrawalSuccessCount:     withdrawalCount, WithdrawalSuccessAmount: withdrawalAmount,
		ExpectedBalance: expectedBalance, GatewayWalletBalance: gatewayBalance,
		DiscrepancyAmount:       discrepancy,
		SettlementPendingAmount: settlementPending, SettlementMinutes: settlementMinutes,
		HasAnomaly: hasAnomaly, GatewayError: gatewayErrMsg, CheckedDate: now,
	}, nil
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
