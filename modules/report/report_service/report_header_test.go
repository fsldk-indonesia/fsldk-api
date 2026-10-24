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
