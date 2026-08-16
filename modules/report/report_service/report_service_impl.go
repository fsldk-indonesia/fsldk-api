package report_service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"time"

	"fsldk-api/base/apperror"
	"fsldk-api/modules/report/report_dto"
	"fsldk-api/modules/report/report_repository"
	"fsldk-api/modules/submission_form/submission_form_repository"
	"fsldk-api/pkg/auditlog"

	"github.com/xuri/excelize/v2"
)

var reportColumns = []string{"Nama Organisasi", "Provinsi", "Kota/Kabupaten", "Status", "Level", "Tanggal Submit", "Terakhir Diperbarui"}

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct {
	repo     report_repository.Repository
	formRepo submission_form_repository.Repository
	orgScope OrgScopeResolver
	audit    *auditlog.Logger
}

// NewService membuat Service report.
func NewService(repo report_repository.Repository, formRepo submission_form_repository.Repository, orgScope OrgScopeResolver, audit *auditlog.Logger) Service {
	return &ServiceImpl{repo: repo, formRepo: formRepo, orgScope: orgScope, audit: audit}
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
	orgIDs, err := s.orgScope.AccessibleOrganizationIDs(ctx, caller.OrganizationID, caller.OrganizationTypeCode, caller.WildcardTierAccess)
	if err != nil {
		return report_dto.ExportResult{}, apperror.Internal("")
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
