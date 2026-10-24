package report_service

import (
	"strings"
	"testing"

	"fsldk-api/modules/report/report_dto"
)

func TestBuildReportHeaderRows_IncludesOnlyAppliedFilters(t *testing.T) {
	rows := buildReportHeaderRows("Laporan Levelisasi", "Portal Puskomda — Banten Raya", report_dto.ReportFilter{
		Statuses: []string{"SUBMITTED", "PUBLISHED"},
	}, "dev1@fsldk.or.id")

	if len(rows) < 7 {
		t.Fatalf("expected at least 7 header rows, got %d", len(rows))
	}
	if rows[0][0] != "Laporan Levelisasi" {
		t.Errorf("row 0 = %v, want title in [0][0]", rows[0])
	}
	if rows[1][0] != "Portal Puskomda — Banten Raya" {
		t.Errorf("row 1 = %v, want portal label in [0][0]", rows[1])
	}

	joined := ""
	for _, r := range rows {
		joined += strings.Join(r, " ") + "\n"
	}
	if !strings.Contains(joined, "dev1@fsldk.or.id") {
		t.Errorf("header rows missing generatedByEmail, got:\n%s", joined)
	}
	if !strings.Contains(joined, "Status: SUBMITTED, PUBLISHED") {
		t.Errorf("header rows missing applied Status filter, got:\n%s", joined)
	}
	if strings.Contains(joined, "Provinsi:") {
		t.Errorf("header rows should NOT mention Provinsi when that filter is empty, got:\n%s", joined)
	}
}

func TestBuildReportHeaderRows_NoFiltersAppliedShowsNeutralPeriod(t *testing.T) {
	rows := buildReportHeaderRows("Laporan Pendataan", "Portal Puskomnas — Nasional", report_dto.ReportFilter{}, "a@b.co")

	joined := ""
	for _, r := range rows {
		joined += strings.Join(r, " ") + "\n"
	}
	if !strings.Contains(joined, "Semua periode") {
		t.Errorf("expected neutral 'Semua periode' when DateFrom/DateTo empty, got:\n%s", joined)
	}
}

// TestRowValues_TranslatesStatusLabel mengunci perbaikan: Excel export
// (toCSV/rowsToValues, keduanya lewat rowValues) harus menampilkan label
// Indonesia yang sama dengan tabel on-screen (SUBMISSION_STATUS_LABELS di
// frontend), bukan kode mentah tr_submission.status — DL-5 "Excel export
// matches exactly what the on-screen table shows".
func TestRowValues_TranslatesStatusLabel(t *testing.T) {
	values := rowValues(report_dto.SubmissionRow{OrganizationName: "LDK Syahid", Status: "PUSKOMDA_REVIEW"})

	const statusColumn = 4 // reportColumns: Nama LDK, Puskomda, Provinsi, Kota/Kabupaten, Status, ...
	if values[statusColumn] != "Verifikasi Puskomda" {
		t.Errorf("rowValues()[%d] = %q, want translated label %q, got raw code instead", statusColumn, values[statusColumn], "Verifikasi Puskomda")
	}
}

// TestReportStatusLabel_FallsBackToRawCodeForUnknownStatus memastikan status
// yang belum masuk peta tetap tampil (fallback ke kode mentah), bukan string
// kosong — pola sama dengan statusLabel() di report.levelisasi.page.ts.
func TestReportStatusLabel_FallsBackToRawCodeForUnknownStatus(t *testing.T) {
	if got := reportStatusLabel("SOME_FUTURE_STATUS"); got != "SOME_FUTURE_STATUS" {
		t.Errorf("reportStatusLabel(unknown) = %q, want fallback to the raw code", got)
	}
}
