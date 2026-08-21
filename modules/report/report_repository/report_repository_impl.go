package report_repository

import (
	"context"

	"fsldk-api/modules/report/report_dto"

	"gorm.io/gorm"
)

// RepositoryImpl adalah implementasi Repository berbasis GORM.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) SubmissionRows(ctx context.Context, formID int64, status string, organizationIDs []int64) ([]report_dto.SubmissionRow, error) {
	if len(organizationIDs) == 0 {
		return []report_dto.SubmissionRow{}, nil
	}
	q := r.db.WithContext(ctx).Table("tr_submission s").
		Select("o.organizationName, o.provinceName, o.cityName, s.status, l.levelLabel, s.submittedDate, s.lastUpdatedDate").
		Joins("JOIN ms_organization o ON o.organizationID = s.organizationID").
		Joins("LEFT JOIN tr_levelisasi_result res ON res.organizationID = s.organizationID AND res.isCurrent = 1").
		Joins("LEFT JOIN lk_level l ON l.levelCode = res.levelCode").
		Where("s.formID = ? AND s.organizationID IN ?", formID, organizationIDs)
	if status != "" {
		q = q.Where("s.status = ?", status)
	}
	var out []report_dto.SubmissionRow
	err := q.Order("o.organizationName ASC").Find(&out).Error
	return out, err
}
