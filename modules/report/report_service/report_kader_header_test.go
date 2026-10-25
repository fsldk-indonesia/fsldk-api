package report_service

import (
	"strings"
	"testing"

	"fsldk-api/modules/report/report_dto"
)

func TestReportKaderStatusLabel_MatchesFrontendKaderStatusLabels(t *testing.T) {
	// Mirror fsldk-web/src/app/modules/submission/entities/submission.ts:42-44
	// KADER_STATUS_LABELS verbatim — Show Data (frontend) dan Download Data
	// (backend Excel) harus menampilkan label yang SAMA PERSIS.
	cases := map[string]string{
		"PENDING": "Menunggu Persetujuan", "ACTIVE": "Aktif", "REJECTED": "Ditolak", "INACTIVE": "Nonaktif",
	}
	for code, want := range cases {
		if got := reportKaderStatusLabel(code); got != want {
			t.Errorf("reportKaderStatusLabel(%q) = %q, want %q", code, got, want)
		}
	}
}

func TestReportKaderStatusLabel_FallsBackToRawCodeForUnknownStatus(t *testing.T) {
	if got := reportKaderStatusLabel("SOME_FUTURE_STATUS"); got != "SOME_FUTURE_STATUS" {
		t.Errorf("reportKaderStatusLabel(unknown) = %q, want fallback to the raw code", got)
	}
}

func TestKaderRowValues_TranslatesStatusLabel(t *testing.T) {
	name := "Budi Santoso"
	values := kaderRowValues(report_dto.KaderReportRow{FullName: name, OrganizationName: "LDK Syahid", Status: "PENDING"})

	const statusColumn = 5 // kaderReportColumns: Nama Lengkap, Nama LDK, Puskomda, Provinsi, Kota/Kabupaten, Status Keanggotaan, ...
	if values[statusColumn] != "Menunggu Persetujuan" {
		t.Errorf("kaderRowValues()[%d] = %q, want translated label %q", statusColumn, values[statusColumn], "Menunggu Persetujuan")
	}
	if values[0] != name {
		t.Errorf("kaderRowValues()[0] = %q, want FullName %q", values[0], name)
	}
}

func TestBuildKaderReportHeaderRows_ShowsLdkNamesNotIDs(t *testing.T) {
	rows := buildKaderReportHeaderRows("Laporan Sensus Kader", report_dto.KaderFilter{
		OrganizationIDs: []int64{7},
		Statuses:        []string{"PENDING", "ACTIVE"},
	}, "dev1@fsldk.or.id", map[int64]string{7: "LDK Syahid"})

	joined := ""
	for _, r := range rows {
		joined += strings.Join(r, " ") + "\n"
	}
	if !strings.Contains(joined, "Nama LDK: LDK Syahid") {
		t.Errorf("header rows should show resolved LDK name, got:\n%s", joined)
	}
	if !strings.Contains(joined, "Status Keanggotaan: Menunggu Persetujuan, Aktif") {
		t.Errorf("header rows should show translated Kader status labels, got:\n%s", joined)
	}
	if rows[0][0] != "Laporan Sensus Kader" {
		t.Errorf("row 0 = %v, want title in [0][0]", rows[0])
	}
}
