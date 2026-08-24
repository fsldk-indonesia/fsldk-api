package jobqueue_repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"fsldk-api/modules/jobqueue/jobqueue_dto"
	"fsldk-api/modules/jobqueue/jobqueue_model"

	"gorm.io/gorm"
)

// RepositoryImpl adalah implementasi Repository berbasis GORM.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) Create(ctx context.Context, j jobqueue_model.Job) (int64, error) {
	values := map[string]interface{}{
		"queue": j.Queue, "jobType": j.JobType, "payload": j.Payload,
		"status": jobqueue_model.StatusPending, "maxAttempts": j.MaxAttempts,
		"availableDate": j.AvailableDate, "createdDate": time.Now(),
	}
	if j.CorrelationType != nil {
		values["correlationType"] = *j.CorrelationType
		values["correlationID"] = j.CorrelationID
	}
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("tr_job_queue").Create(values).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) FindByID(ctx context.Context, id int64) (jobqueue_model.Job, error) {
	var j jobqueue_model.Job
	err := r.db.WithContext(ctx).Table("tr_job_queue").Where("jobID = ?", id).Take(&j).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return jobqueue_model.Job{}, ErrNotFound
	}
	return j, err
}

func (r *RepositoryImpl) List(ctx context.Context, f jobqueue_dto.ListFilter) ([]jobqueue_model.Job, int64, error) {
	base := r.db.WithContext(ctx).Table("tr_job_queue")
	if f.Status != "" {
		base = base.Where("status = ?", f.Status)
	}
	if f.Queue != "" {
		base = base.Where("queue = ?", f.Queue)
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		base = base.Where("(jobType LIKE ? OR lastError LIKE ?)", like, like)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	orderBy := f.OrderBy
	if orderBy == "" {
		orderBy = "createdDate DESC"
	}
	var out []jobqueue_model.Job
	err := base.Order(orderBy).Limit(f.Limit).Offset(f.Offset).Find(&out).Error
	return out, total, err
}

func (r *RepositoryImpl) Stats(ctx context.Context, stuckThreshold time.Duration) (jobqueue_dto.StatsResponse, error) {
	var stats jobqueue_dto.StatsResponse
	now := time.Now()
	stuckCutoff := now.Add(-stuckThreshold)
	base := r.db.WithContext(ctx).Table("tr_job_queue")

	if err := base.Session(&gorm.Session{}).Where("status = ? AND availableDate <= ?", jobqueue_model.StatusPending, now).Count(&stats.Pending).Error; err != nil {
		return stats, err
	}
	if err := base.Session(&gorm.Session{}).Where("status = ? AND availableDate > ?", jobqueue_model.StatusPending, now).Count(&stats.Delayed).Error; err != nil {
		return stats, err
	}
	if err := base.Session(&gorm.Session{}).Where("status = ? AND reservedDate >= ?", jobqueue_model.StatusProcessing, stuckCutoff).Count(&stats.Processing).Error; err != nil {
		return stats, err
	}
	if err := base.Session(&gorm.Session{}).Where("status = ? AND reservedDate < ?", jobqueue_model.StatusProcessing, stuckCutoff).Count(&stats.Stuck).Error; err != nil {
		return stats, err
	}
	if err := base.Session(&gorm.Session{}).Where("status = ?", jobqueue_model.StatusFailed).Count(&stats.Failed).Error; err != nil {
		return stats, err
	}
	if err := base.Session(&gorm.Session{}).Where("status = ?", jobqueue_model.StatusCompleted).Count(&stats.Completed).Error; err != nil {
		return stats, err
	}
	return stats, nil
}

func (r *RepositoryImpl) Claim(ctx context.Context, queue string) (jobqueue_model.Job, bool, error) {
	var job jobqueue_model.Job
	err := r.db.WithContext(ctx).Table("tr_job_queue").
		Where("status = ? AND queue = ? AND availableDate <= ?", jobqueue_model.StatusPending, queue, time.Now()).
		Order("availableDate ASC").Limit(1).Take(&job).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return jobqueue_model.Job{}, false, nil
	}
	if err != nil {
		return jobqueue_model.Job{}, false, err
	}

	result := r.db.WithContext(ctx).Table("tr_job_queue").
		Where("jobID = ? AND status = ?", job.JobID, jobqueue_model.StatusPending).
		Updates(map[string]interface{}{
			"status":       jobqueue_model.StatusProcessing,
			"reservedDate": time.Now(),
			"attempts":     gorm.Expr("attempts + 1"),
		})
	if result.Error != nil {
		return jobqueue_model.Job{}, false, result.Error
	}
	if result.RowsAffected == 0 {
		return jobqueue_model.Job{}, false, nil // worker/instance lain sudah mengklaim duluan
	}
	job.Status = jobqueue_model.StatusProcessing
	job.Attempts++
	return job, true, nil
}

func (r *RepositoryImpl) MarkCompleted(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Table("tr_job_queue").Where("jobID = ?", id).
		Updates(map[string]interface{}{"status": jobqueue_model.StatusCompleted, "completedDate": time.Now()}).Error
}

func (r *RepositoryImpl) MarkRetryOrFail(ctx context.Context, id int64, lastErr string, nextAvailable *time.Time) error {
	values := map[string]interface{}{"lastError": lastErr}
	if nextAvailable != nil {
		values["status"] = jobqueue_model.StatusPending
		values["reservedDate"] = nil
		values["availableDate"] = *nextAvailable
	} else {
		values["status"] = jobqueue_model.StatusFailed
		values["failedDate"] = time.Now()
	}
	return r.db.WithContext(ctx).Table("tr_job_queue").Where("jobID = ?", id).Updates(values).Error
}

func (r *RepositoryImpl) SweepStuck(ctx context.Context, threshold time.Duration) (int64, int64, error) {
	cutoff := time.Now().Add(-threshold)

	recycle := r.db.WithContext(ctx).Table("tr_job_queue").
		Where("status = ? AND reservedDate < ? AND attempts < maxAttempts", jobqueue_model.StatusProcessing, cutoff).
		Updates(map[string]interface{}{"status": jobqueue_model.StatusPending, "reservedDate": nil, "availableDate": time.Now()})
	if recycle.Error != nil {
		return 0, 0, recycle.Error
	}

	fail := r.db.WithContext(ctx).Table("tr_job_queue").
		Where("status = ? AND reservedDate < ? AND attempts >= maxAttempts", jobqueue_model.StatusProcessing, cutoff).
		Updates(map[string]interface{}{
			"status": jobqueue_model.StatusFailed, "failedDate": time.Now(),
			"lastError": "stuck: exceeded max attempts during recovery",
		})
	if fail.Error != nil {
		return recycle.RowsAffected, 0, fail.Error
	}
	return recycle.RowsAffected, fail.RowsAffected, nil
}

func (r *RepositoryImpl) Retry(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).Table("tr_job_queue").
		Where("jobID = ? AND status = ?", id, jobqueue_model.StatusFailed).
		Updates(map[string]interface{}{
			"status": jobqueue_model.StatusPending, "attempts": 0,
			"lastError": nil, "failedDate": nil, "availableDate": time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrInvalidState
	}
	return nil
}

func (r *RepositoryImpl) Delete(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).Exec(
		"DELETE FROM tr_job_queue WHERE jobID = ? AND status IN (?, ?)",
		id, jobqueue_model.StatusFailed, jobqueue_model.StatusCompleted,
	)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrInvalidState
	}
	return nil
}

func (r *RepositoryImpl) LogOutboundMessage(ctx context.Context, jobID int64, waMessageID, toPhone, templateName, correlationType string, correlationID int64) error {
	values := map[string]interface{}{
		"jobID": jobID, "waMessageID": waMessageID, "toPhone": toPhone,
		"templateName": templateName, "createdDate": time.Now(),
	}
	if correlationType != "" {
		values["correlationType"] = correlationType
		values["correlationID"] = correlationID
	}
	return r.db.WithContext(ctx).Table("tr_whatsapp_message_log").Create(values).Error
}

func (r *RepositoryImpl) ResolveCorrelation(ctx context.Context, waMessageID string) (string, int64, bool, error) {
	var row struct {
		CorrelationType sql.NullString
		CorrelationID   sql.NullInt64
	}
	err := r.db.WithContext(ctx).Table("tr_whatsapp_message_log").
		Select("correlationType, correlationID").Where("waMessageID = ?", waMessageID).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", 0, false, nil
	}
	if err != nil {
		return "", 0, false, err
	}
	if !row.CorrelationType.Valid || !row.CorrelationID.Valid {
		return "", 0, false, nil
	}
	return row.CorrelationType.String, row.CorrelationID.Int64, true, nil
}

func (r *RepositoryImpl) FindRecentByPhone(ctx context.Context, phone, correlationType string, limit int) ([]int64, error) {
	var ids []int64
	err := r.db.WithContext(ctx).Table("tr_whatsapp_message_log").
		Where("toPhone = ? AND correlationType = ?", phone, correlationType).
		Order("createdDate DESC").Limit(limit).Pluck("correlationID", &ids).Error
	return ids, err
}

func (r *RepositoryImpl) FindJobIDByWAMessageID(ctx context.Context, waMessageID string) (int64, bool, error) {
	var row struct {
		JobID sql.NullInt64
	}
	err := r.db.WithContext(ctx).Table("tr_whatsapp_message_log").
		Select("jobID").Where("waMessageID = ?", waMessageID).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	if !row.JobID.Valid {
		return 0, false, nil
	}
	return row.JobID.Int64, true, nil
}

func (r *RepositoryImpl) RecordDeliveryFailure(ctx context.Context, jobID int64, note string) error {
	return r.db.WithContext(ctx).Table("tr_job_queue").Where("jobID = ?", jobID).
		Update("lastError", note).Error
}

func (r *RepositoryImpl) UpdateWAMessageID(ctx context.Context, oldID, newID string) error {
	if oldID == "" || newID == "" || oldID == newID {
		return nil
	}
	return r.db.WithContext(ctx).Table("tr_whatsapp_message_log").Where("waMessageID = ?", oldID).
		Update("waMessageID", newID).Error
}
