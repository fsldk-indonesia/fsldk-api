package submission_repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"fsldk-api/modules/submission/submission_dto"
	"fsldk-api/modules/submission/submission_model"

	"gorm.io/gorm"
)

// RepositoryImpl adalah implementasi Repository berbasis GORM.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) Create(ctx context.Context, p submission_model.CreateParams) (int64, error) {
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("tr_submission").Create(map[string]interface{}{
			"formID":            p.FormID,
			"formVersionID":     p.FormVersionID,
			"organizationID":    p.OrganizationID,
			"subjectType":       p.SubjectType,
			"submittedByUserID": p.SubmittedByUserID,
			"status":            "DRAFT",
			"version":           1,
			"createdDate":       time.Now(),
			"createdBy":         p.CreatedBy,
		}).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) FindByID(ctx context.Context, id int64) (submission_model.Submission, error) {
	var s submission_model.Submission
	err := r.db.WithContext(ctx).Table("tr_submission").Where("submissionID = ?", id).Take(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return submission_model.Submission{}, ErrNotFound
	}
	return s, err
}

func (r *RepositoryImpl) FindByOwnerOrganization(ctx context.Context, organizationID, formID int64) (submission_model.Submission, error) {
	var s submission_model.Submission
	err := r.db.WithContext(ctx).Table("tr_submission").
		Where("organizationID = ? AND formID = ? AND subjectType = 'ORGANIZATION'", organizationID, formID).
		Take(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return submission_model.Submission{}, ErrNotFound
	}
	return s, err
}

func (r *RepositoryImpl) FindByUserAndForm(ctx context.Context, userID, formID int64) (submission_model.Submission, error) {
	var s submission_model.Submission
	err := r.db.WithContext(ctx).Table("tr_submission").
		Where("submittedByUserID = ? AND formID = ? AND subjectType = 'KADER'", userID, formID).
		Take(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return submission_model.Submission{}, ErrNotFound
	}
	return s, err
}

func (r *RepositoryImpl) List(ctx context.Context, f submission_dto.ListFilter) ([]submission_model.Submission, int64, error) {
	base := r.db.WithContext(ctx).Table("tr_submission")
	if f.SubmittedByUserID != nil {
		base = base.Where("submittedByUserID = ?", *f.SubmittedByUserID)
	} else {
		if len(f.OrganizationIDs) == 0 {
			return []submission_model.Submission{}, 0, nil
		}
		base = base.Where("organizationID IN ?", f.OrganizationIDs)
	}
	if f.Status != "" {
		base = base.Where("status = ?", f.Status)
	}
	if f.FormID > 0 {
		base = base.Where("formID = ?", f.FormID)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	orderBy := f.OrderBy
	if orderBy == "" {
		orderBy = "createdDate DESC"
	}
	var out []submission_model.Submission
	q := base.Order(orderBy)
	if f.Limit > 0 {
		q = q.Limit(f.Limit).Offset(f.Offset)
	}
	err := q.Find(&out).Error
	return out, total, err
}

func (r *RepositoryImpl) UpdateStatus(ctx context.Context, id int64, toStatus string, submittedDate sql.NullTime, updatedBy int64) error {
	values := map[string]interface{}{
		"status":          toStatus,
		"version":         gorm.Expr("version + 1"),
		"lastUpdatedDate": time.Now(),
		"updatedDate":     time.Now(),
		"updatedBy":       updatedBy,
	}
	if submittedDate.Valid {
		values["submittedDate"] = submittedDate
	}
	return r.db.WithContext(ctx).Table("tr_submission").Where("submissionID = ?", id).Updates(values).Error
}

func (r *RepositoryImpl) UpdateFormVersion(ctx context.Context, id int64, formVersionID int64) error {
	return r.db.WithContext(ctx).Table("tr_submission").Where("submissionID = ?", id).
		Update("formVersionID", formVersionID).Error
}

func (r *RepositoryImpl) TouchVersion(ctx context.Context, id int64, updatedBy int64) error {
	return r.db.WithContext(ctx).Table("tr_submission").Where("submissionID = ?", id).Updates(map[string]interface{}{
		"version":         gorm.Expr("version + 1"),
		"lastUpdatedDate": time.Now(),
		"updatedDate":     time.Now(),
		"updatedBy":       updatedBy,
	}).Error
}

func (r *RepositoryImpl) SetSubjectReference(ctx context.Context, id int64, subjectReferenceID int64) error {
	return r.db.WithContext(ctx).Table("tr_submission").Where("submissionID = ?", id).
		Update("subjectReferenceID", subjectReferenceID).Error
}

func (r *RepositoryImpl) UpsertAnswer(ctx context.Context, submissionID int64, p submission_model.AnswerParams) error {
	return r.db.WithContext(ctx).Exec(
		`INSERT INTO tr_submission_answer
			(submissionID, fieldID, valueText, valueNumber, valueDate, valueOptionID, valueFileURL, valueFileName)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
			valueText = VALUES(valueText), valueNumber = VALUES(valueNumber), valueDate = VALUES(valueDate),
			valueOptionID = VALUES(valueOptionID), valueFileURL = VALUES(valueFileURL), valueFileName = VALUES(valueFileName)`,
		submissionID, p.FieldID, p.ValueText, p.ValueNumber, p.ValueDate, p.ValueOptionID, p.ValueFileURL, p.ValueFileName,
	).Error
}

func (r *RepositoryImpl) RemapAnswerField(ctx context.Context, submissionID, oldFieldID, newFieldID int64) error {
	return r.db.WithContext(ctx).Table("tr_submission_answer").
		Where("submissionID = ? AND fieldID = ?", submissionID, oldFieldID).
		Update("fieldID", newFieldID).Error
}

func (r *RepositoryImpl) ListAnswersBySubmission(ctx context.Context, submissionID int64) ([]submission_model.Answer, error) {
	var out []submission_model.Answer
	err := r.db.WithContext(ctx).Table("tr_submission_answer").
		Where("submissionID = ?", submissionID).Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) InsertStatusHistory(ctx context.Context, submissionID int64, fromStatus sql.NullString, toStatus string, actorUserID int64, actorOrganizationID sql.NullInt64, note sql.NullString) error {
	return r.db.WithContext(ctx).Table("tr_submission_status_history").Create(map[string]interface{}{
		"submissionID":        submissionID,
		"fromStatus":          fromStatus,
		"toStatus":            toStatus,
		"actorUserID":         actorUserID,
		"actorOrganizationID": actorOrganizationID,
		"note":                note,
		"createdDate":         time.Now(),
	}).Error
}

func (r *RepositoryImpl) ListStatusHistory(ctx context.Context, submissionID int64) ([]submission_model.StatusHistory, error) {
	var out []submission_model.StatusHistory
	err := r.db.WithContext(ctx).Table("tr_submission_status_history").
		Where("submissionID = ?", submissionID).Order("createdDate ASC").Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) CreateKader(ctx context.Context, p submission_model.KaderParams) (int64, error) {
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("ms_kader").Create(map[string]interface{}{
			"submissionID":   p.SubmissionID,
			"organizationID": p.OrganizationID,
			"userID":         p.UserID,
			"fullName":       p.FullName,
			"status":         "PENDING",
			"createdDate":    time.Now(),
		}).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) FindKaderBySubmission(ctx context.Context, submissionID int64) (submission_model.Kader, error) {
	var k submission_model.Kader
	err := r.db.WithContext(ctx).Table("ms_kader").Where("submissionID = ?", submissionID).Take(&k).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return submission_model.Kader{}, ErrNotFound
	}
	return k, err
}

func (r *RepositoryImpl) FindKaderByID(ctx context.Context, id int64) (submission_model.Kader, error) {
	var k submission_model.Kader
	err := r.db.WithContext(ctx).Table("ms_kader").Where("kaderID = ?", id).Take(&k).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return submission_model.Kader{}, ErrNotFound
	}
	return k, err
}

func (r *RepositoryImpl) UpdateKaderStatus(ctx context.Context, id int64, status string, uniqueCode sql.NullString, issuedDate sql.NullTime) error {
	values := map[string]interface{}{
		"status":      status,
		"updatedDate": time.Now(),
	}
	if uniqueCode.Valid {
		values["uniqueCode"] = uniqueCode
	}
	if issuedDate.Valid {
		values["issuedDate"] = issuedDate
	}
	return r.db.WithContext(ctx).Table("ms_kader").Where("kaderID = ?", id).Updates(values).Error
}

func (r *RepositoryImpl) ListKadersByOrganizations(ctx context.Context, organizationIDs []int64, status string) ([]submission_model.Kader, error) {
	q := r.db.WithContext(ctx).Table("ms_kader").Where("organizationID IN ?", organizationIDs)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var out []submission_model.Kader
	err := q.Order("createdDate DESC").Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) CountKaderCodesForOrgYear(ctx context.Context, codePrefix string) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("ms_kader").Where("uniqueCode LIKE ?", codePrefix+"%").Count(&count).Error
	return int(count), err
}

func (r *RepositoryImpl) CreateReview(ctx context.Context, p submission_model.ReviewParams) (int64, error) {
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("tr_submission_review").Create(map[string]interface{}{
			"submissionID":   p.SubmissionID,
			"tierLevel":      p.TierLevel,
			"reviewerUserID": p.ReviewerUserID,
			"decision":       p.Decision,
			"checklistJSON":  p.ChecklistJSON,
			"note":           p.Note,
			"reviewedDate":   time.Now(),
		}).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) ListReviewsBySubmission(ctx context.Context, submissionID int64) ([]submission_model.Review, error) {
	var out []submission_model.Review
	err := r.db.WithContext(ctx).Table("tr_submission_review").
		Where("submissionID = ?", submissionID).Order("reviewedDate ASC").Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) FindCurrentLevelResult(ctx context.Context, organizationID int64) (submission_model.LevelisasiResult, error) {
	var res submission_model.LevelisasiResult
	err := r.db.WithContext(ctx).Table("tr_levelisasi_result").
		Where("organizationID = ? AND isCurrent = 1", organizationID).Take(&res).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return submission_model.LevelisasiResult{}, ErrNotFound
	}
	return res, err
}

func (r *RepositoryImpl) FindLevelResultBySubmission(ctx context.Context, submissionID int64) (submission_model.LevelisasiResult, error) {
	var res submission_model.LevelisasiResult
	err := r.db.WithContext(ctx).Table("tr_levelisasi_result").
		Where("submissionID = ? AND isCurrent = 1", submissionID).Take(&res).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return submission_model.LevelisasiResult{}, ErrNotFound
	}
	return res, err
}

func (r *RepositoryImpl) ClearCurrentLevelResult(ctx context.Context, organizationID int64) error {
	return r.db.WithContext(ctx).Table("tr_levelisasi_result").
		Where("organizationID = ? AND isCurrent = 1", organizationID).
		Update("isCurrent", false).Error
}

func (r *RepositoryImpl) CreateLevelResult(ctx context.Context, submissionID, organizationID int64, levelCode, justificationNote string, establishedByUserID int64) (int64, error) {
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("tr_levelisasi_result").Create(map[string]interface{}{
			"submissionID":        submissionID,
			"organizationID":      organizationID,
			"levelCode":           levelCode,
			"justificationNote":   sql.NullString{String: justificationNote, Valid: justificationNote != ""},
			"establishedByUserID": establishedByUserID,
			"establishedDate":     time.Now(),
			"isPublished":         false,
			"isCurrent":           true,
		}).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) UpdateLevelResult(ctx context.Context, resultID int64, levelCode, justificationNote string, establishedByUserID int64) error {
	return r.db.WithContext(ctx).Table("tr_levelisasi_result").Where("resultID = ?", resultID).Updates(map[string]interface{}{
		"levelCode":           levelCode,
		"justificationNote":   sql.NullString{String: justificationNote, Valid: justificationNote != ""},
		"establishedByUserID": establishedByUserID,
		"establishedDate":     time.Now(),
	}).Error
}

func (r *RepositoryImpl) PublishLevelResult(ctx context.Context, resultID int64) error {
	return r.db.WithContext(ctx).Table("tr_levelisasi_result").Where("resultID = ?", resultID).Updates(map[string]interface{}{
		"isPublished":   true,
		"publishedDate": time.Now(),
	}).Error
}

func (r *RepositoryImpl) LevelLabel(ctx context.Context, levelCode string) (string, error) {
	var label string
	err := r.db.WithContext(ctx).Table("lk_level").Select("levelLabel").Where("levelCode = ?", levelCode).Take(&label).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", ErrNotFound
	}
	return label, err
}
