package schedule_repository

import (
	"context"
	"errors"
	"time"

	"fsldk-api/constants"
	"fsldk-api/modules/schedule/schedule_dto"
	"fsldk-api/modules/schedule/schedule_model"

	"gorm.io/gorm"
)

const selectCols = "s.scheduleID, s.title, s.description, s.category, s.startDate, s.endDate, " +
	"s.isAllDay, s.startTime, s.endTime, s.location, s.organizer, s.contactPerson, " +
	"s.url, s.isActive, s.createdDate, s.updatedDate"

// RepositoryImpl is the GORM-based Repository implementation.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository creates the Repository implementation.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) baseQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Table(constants.TableSchedule + " s")
}

func (r *RepositoryImpl) List(ctx context.Context, f schedule_dto.Filter) ([]schedule_model.Schedule, int64, error) {
	q := r.baseQuery(ctx)
	if f.ActiveOnly {
		q = q.Where("s.isActive = 1")
	}
	if f.Search != "" {
		q = q.Where("s.title LIKE ?", "%"+f.Search+"%")
	}
	if f.Category != "" {
		q = q.Where("s.category = ?", f.Category)
	}
	if f.Month > 0 {
		q = q.Where("MONTH(s.startDate) = ?", f.Month)
	}
	if f.Year > 0 {
		q = q.Where("YEAR(s.startDate) = ?", f.Year)
	}
	// Overlap window: an activity is in range when it starts on/before DateTo
	// and ends (endDate, or startDate when single-day) on/after DateFrom.
	if f.DateFrom != "" {
		q = q.Where("COALESCE(s.endDate, s.startDate) >= ?", f.DateFrom)
	}
	if f.DateTo != "" {
		q = q.Where("s.startDate <= ?", f.DateTo)
	}

	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	q = q.Select(selectCols).Order(f.OrderBy)
	if f.Limit > 0 {
		q = q.Limit(f.Limit).Offset(f.Offset)
	}

	var out []schedule_model.Schedule
	err := q.Find(&out).Error
	return out, total, err
}

func (r *RepositoryImpl) FindByID(ctx context.Context, id int64) (schedule_model.Schedule, error) {
	var m schedule_model.Schedule
	err := r.baseQuery(ctx).Select(selectCols).Where("s.scheduleID = ?", id).Take(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return schedule_model.Schedule{}, ErrNotFound
	}
	return m, err
}

func (r *RepositoryImpl) Create(ctx context.Context, s schedule_model.Schedule, actorID int64) (int64, error) {
	values := writeValues(s)
	values["isActive"] = true
	values["createdDate"] = time.Now()
	values["createdBy"] = actorID

	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table(constants.TableSchedule).Create(values).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) Update(ctx context.Context, id int64, s schedule_model.Schedule, actorID int64) error {
	values := writeValues(s)
	values["updatedDate"] = time.Now()
	values["updatedBy"] = actorID
	return r.db.WithContext(ctx).Table(constants.TableSchedule).Where("scheduleID = ?", id).Updates(values).Error
}

func (r *RepositoryImpl) SetActive(ctx context.Context, id int64, isActive bool, actorID int64) error {
	res := r.db.WithContext(ctx).Table(constants.TableSchedule).Where("scheduleID = ?", id).Updates(map[string]interface{}{
		"isActive":    isActive,
		"updatedDate": time.Now(),
		"updatedBy":   actorID,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *RepositoryImpl) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Exec("DELETE FROM "+constants.TableSchedule+" WHERE scheduleID = ?", id).Error
}

// writeValues maps model fields to a column-value map for GORM writes. Nil
// pointers become SQL NULL, which is what clearing an optional field on edit
// should do.
func writeValues(s schedule_model.Schedule) map[string]interface{} {
	return map[string]interface{}{
		"title":         s.Title,
		"description":   s.Description,
		"category":      s.Category,
		"startDate":     s.StartDate,
		"endDate":       s.EndDate,
		"isAllDay":      s.IsAllDay,
		"startTime":     s.StartTime,
		"endTime":       s.EndTime,
		"location":      s.Location,
		"organizer":     s.Organizer,
		"contactPerson": s.ContactPerson,
		"url":           s.URL,
	}
}
