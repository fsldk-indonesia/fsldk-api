package dashboard_repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"fsldk-api/modules/dashboard/dashboard_dto"

	"gorm.io/gorm"
)

// RepositoryImpl adalah implementasi Repository berbasis GORM.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

const statusBucketCase = `CASE
	WHEN s.status IS NULL OR s.status = 'DRAFT' THEN 'BELUM_MENGISI'
	WHEN s.status IN ('SUBMITTED','PUSKOMDA_REVIEW') THEN 'MENUNGGU_VERIFIKASI'
	WHEN s.status IN ('REVISION_REQUESTED_PUSKOMDA','REVISION_REQUESTED_PUSKOMNAS') THEN 'PERLU_REVISI'
	ELSE 'TERVERIFIKASI'
END`

func (r *RepositoryImpl) LDKSummary(ctx context.Context, organizationID int64, levelisasiFormID int64) (dashboard_dto.LDKSummary, error) {
	var out dashboard_dto.LDKSummary

	var sub struct {
		Status          string
		LastUpdatedDate sql.NullTime
	}
	err := r.db.WithContext(ctx).Table("tr_submission").
		Select("status, lastUpdatedDate").
		Where("organizationID = ? AND formID = ?", organizationID, levelisasiFormID).
		Take(&sub).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		out.SubmissionStatus = "BELUM_MENGISI"
	case err != nil:
		return out, err
	default:
		out.SubmissionStatus = sub.Status
		if sub.LastUpdatedDate.Valid {
			out.LastUpdatedDate = &sub.LastUpdatedDate.Time
		}
	}

	var level struct {
		LevelCode  string
		LevelLabel string
	}
	if err := r.db.WithContext(ctx).Table("tr_levelisasi_result res").
		Select("res.levelCode, l.levelLabel").
		Joins("JOIN lk_level l ON l.levelCode = res.levelCode").
		Where("res.organizationID = ? AND res.isCurrent = 1", organizationID).
		Take(&level).Error; err == nil {
		out.LevelCode = level.LevelCode
		out.LevelLabel = level.LevelLabel
	}

	var pending, active int64
	r.db.WithContext(ctx).Table("ms_kader").Where("organizationID = ? AND status = 'PENDING'", organizationID).Count(&pending)
	r.db.WithContext(ctx).Table("ms_kader").Where("organizationID = ? AND status = 'ACTIVE'", organizationID).Count(&active)
	out.KaderPending = int(pending)
	out.KaderActive = int(active)

	var notes []struct {
		Note        sql.NullString
		CreatedDate time.Time
	}
	r.db.WithContext(ctx).Table("tr_submission_status_history h").
		Select("h.note, h.createdDate").
		Joins("JOIN tr_submission s ON s.submissionID = h.submissionID").
		Where("s.organizationID = ? AND h.note IS NOT NULL AND h.note <> ''", organizationID).
		Order("h.createdDate DESC").Limit(5).Find(&notes)
	out.RecentNotes = make([]dashboard_dto.NoteEntry, 0, len(notes))
	for _, n := range notes {
		out.RecentNotes = append(out.RecentNotes, dashboard_dto.NoteEntry{Note: n.Note.String, CreatedDate: n.CreatedDate})
	}

	return out, nil
}

func (r *RepositoryImpl) StatusBuckets(ctx context.Context, levelisasiFormID int64, parentOrganizationID *int64) (dashboard_dto.StatusCounts, error) {
	var out dashboard_dto.StatusCounts
	q := r.db.WithContext(ctx).Table("ms_organization o").
		Select(statusBucketCase+" AS bucket, COUNT(*) AS cnt").
		Joins("LEFT JOIN tr_submission s ON s.organizationID = o.organizationID AND s.formID = ?", levelisasiFormID).
		Where("o.organizationTypeCode = 'LDK'")
	if parentOrganizationID != nil {
		q = q.Where("o.parentOrganizationID = ?", *parentOrganizationID)
	}
	var rows []struct {
		Bucket string
		Cnt    int
	}
	if err := q.Group("bucket").Find(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		switch row.Bucket {
		case "BELUM_MENGISI":
			out.BelumMengisi = row.Cnt
		case "MENUNGGU_VERIFIKASI":
			out.MenungguVerifikasi = row.Cnt
		case "PERLU_REVISI":
			out.PerluRevisi = row.Cnt
		case "TERVERIFIKASI":
			out.Terverifikasi = row.Cnt
		}
	}
	return out, nil
}

func (r *RepositoryImpl) CountLDK(ctx context.Context, parentOrganizationID *int64) (int, error) {
	q := r.db.WithContext(ctx).Table("ms_organization").Where("organizationTypeCode = 'LDK'")
	if parentOrganizationID != nil {
		q = q.Where("parentOrganizationID = ?", *parentOrganizationID)
	}
	var count int64
	err := q.Count(&count).Error
	return int(count), err
}

func (r *RepositoryImpl) CountActiveKader(ctx context.Context, parentOrganizationID *int64) (int, error) {
	q := r.db.WithContext(ctx).Table("ms_kader k").
		Joins("JOIN ms_organization o ON o.organizationID = k.organizationID").
		Where("k.status = 'ACTIVE'")
	if parentOrganizationID != nil {
		q = q.Where("o.parentOrganizationID = ?", *parentOrganizationID)
	}
	var count int64
	err := q.Count(&count).Error
	return int(count), err
}

func (r *RepositoryImpl) CountPuskomda(ctx context.Context) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("ms_organization").Where("organizationTypeCode = 'PUSKOMDA'").Count(&count).Error
	return int(count), err
}

func (r *RepositoryImpl) CountLevelEstablished(ctx context.Context, parentOrganizationID *int64) (int, error) {
	q := r.db.WithContext(ctx).Table("tr_levelisasi_result res").
		Joins("JOIN ms_organization o ON o.organizationID = res.organizationID").
		Where("res.isCurrent = 1")
	if parentOrganizationID != nil {
		q = q.Where("o.parentOrganizationID = ?", *parentOrganizationID)
	}
	var count int64
	err := q.Count(&count).Error
	return int(count), err
}

func (r *RepositoryImpl) LevelDistribution(ctx context.Context) ([]dashboard_dto.LevelCount, error) {
	var rows []dashboard_dto.LevelCount
	err := r.db.WithContext(ctx).Table("tr_levelisasi_result res").
		Select("res.levelCode, l.levelLabel, COUNT(*) AS count").
		Joins("JOIN lk_level l ON l.levelCode = res.levelCode").
		Where("res.isCurrent = 1").
		Group("res.levelCode, l.levelLabel").
		Order("l.sortOrder ASC").
		Find(&rows).Error
	return rows, err
}

func (r *RepositoryImpl) PerPuskomdaBreakdown(ctx context.Context) ([]dashboard_dto.PuskomdaBreakdown, error) {
	var rows []dashboard_dto.PuskomdaBreakdown
	err := r.db.WithContext(ctx).Table("ms_organization p").
		Select("p.organizationID, p.organizationName, " +
			"COUNT(DISTINCT o.organizationID) AS totalLDK, " +
			"COUNT(DISTINCT CASE WHEN k.status = 'ACTIVE' THEN k.kaderID END) AS kaderAktif").
		Joins("LEFT JOIN ms_organization o ON o.parentOrganizationID = p.organizationID AND o.organizationTypeCode = 'LDK'").
		Joins("LEFT JOIN ms_kader k ON k.organizationID = o.organizationID").
		Where("p.organizationTypeCode = 'PUSKOMDA'").
		Group("p.organizationID, p.organizationName").
		Order("p.organizationName ASC").
		Find(&rows).Error
	return rows, err
}
