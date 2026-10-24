package report_service

import (
	"strings"
	"testing"

	"fsldk-api/modules/report/report_dto"
)

func TestBuildReportHeaderRows_IncludesOnlyAppliedFilters(t *testing.T) {
	rows := buildReportHeaderRows("Laporan Levelisasi", report_dto.ReportFilter{
		Statuses: []string{"SUBMITTED", "PUBLISHED"},
	}, "dev1@fsldk.or.id", nil)

	if len(rows) < 6 {
		t.Fatalf("expected at least 6 header rows, got %d", len(rows))
	}
	if rows[0][0] != "Laporan Levelisasi" {
		t.Errorf("row 0 = %v, want title in [0][0]", rows[0])
	}
	if len(rows[1]) != 0 {
		t.Errorf("row 1 = %v, want a blank row right after the title (no portal/branch line)", rows[1])
	}

	joined := ""
	for _, r := range rows {
		joined += strings.Join(r, " ") + "\n"
	}
	if !strings.Contains(joined, "dev1@fsldk.or.id") {
		t.Errorf("header rows missing generatedByEmail, got:\n%s", joined)
	}
	if !strings.Contains(joined, "Status: Terkirim, Dipublikasikan") {
		t.Errorf("header rows missing translated applied Status filter, got:\n%s", joined)
	}
	if strings.Contains(joined, "Provinsi:") {
		t.Errorf("header rows should NOT mention Provinsi when that filter is empty, got:\n%s", joined)
	}
}

func TestBuildReportHeaderRows_NoFiltersAppliedShowsNeutralPeriod(t *testing.T) {
	rows := buildReportHeaderRows("Laporan Pendataan", report_dto.ReportFilter{}, "a@b.co", nil)

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

// TestBuildReportHeaderRows_ShowsPuskomdaNamesNotIDs mengunci perbaikan:
// baris "Filter Diterapkan" harus menampilkan NAMA Puskomda (sudah
// di-resolve pemanggil lewat Repository.OrganizationNamesByIDs), bukan ID
// mentah — permintaan user saat manual testing.
func TestBuildReportHeaderRows_ShowsPuskomdaNamesNotIDs(t *testing.T) {
	rows := buildReportHeaderRows("Laporan Levelisasi", report_dto.ReportFilter{
		ParentOrganizationIDs: []int64{12, 45},
	}, "dev1@fsldk.or.id", map[int64]string{12: "Banten Raya", 45: "Jakarta Raya"})

	joined := ""
	for _, r := range rows {
		joined += strings.Join(r, " ") + "\n"
	}
	if !strings.Contains(joined, "Puskomda: Banten Raya, Jakarta Raya") {
		t.Errorf("header rows should show resolved Puskomda names, got:\n%s", joined)
	}
	if strings.Contains(joined, "12, 45") {
		t.Errorf("header rows should NOT show raw Puskomda IDs when names are resolved, got:\n%s", joined)
	}
}

// TestBuildReportHeaderRows_ShowsLdkNames mengunci filter "Nama LDK" (baru)
// — sama pola dengan Puskomda, nama di-resolve lewat peta yang sama
// (resolveReportOrgNames menggabungkan ParentOrganizationIDs+OrganizationIDs
// jadi satu query).
func TestBuildReportHeaderRows_ShowsLdkNames(t *testing.T) {
	rows := buildReportHeaderRows("Laporan Levelisasi", report_dto.ReportFilter{
		OrganizationIDs: []int64{7},
	}, "dev1@fsldk.or.id", map[int64]string{7: "LDK Syahid"})

	joined := ""
	for _, r := range rows {
		joined += strings.Join(r, " ") + "\n"
	}
	if !strings.Contains(joined, "Nama LDK: LDK Syahid") {
		t.Errorf("header rows should show resolved LDK name, got:\n%s", joined)
	}
}

// TestBuildReportHeaderRows_FallsBackToIDsWhenNamesUnresolved memastikan
// informasi filter tidak hilang diam-diam kalau resolusi nama gagal/kosong
// (mis. organisasi sudah terhapus) — fallback ke ID mentah.
func TestBuildReportHeaderRows_FallsBackToIDsWhenNamesUnresolved(t *testing.T) {
	rows := buildReportHeaderRows("Laporan Levelisasi", report_dto.ReportFilter{
		ParentOrganizationIDs: []int64{12},
	}, "dev1@fsldk.or.id", nil)

	joined := ""
	for _, r := range rows {
		joined += strings.Join(r, " ") + "\n"
	}
	if !strings.Contains(joined, "Puskomda: 12") {
		t.Errorf("header rows should fall back to raw ID when no names were resolved, got:\n%s", joined)
	}
}
