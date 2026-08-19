package event_repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"fsldk-api/modules/event/event_dto"
	"fsldk-api/modules/event/event_model"

	"gorm.io/gorm"
)

// selectCols is the column list for event list/detail queries.
const selectCols = "e.eventID, e.eventTitle, e.eventSlug, e.eventDivision, e.eventContent, " +
	"e.eventImage, e.startDate, e.endDate, e.closeRegistDate, e.location, e.place, " +
	"e.locationLink, e.registrationLink, e.documentLink, e.presentationLink, " +
	"e.contactPerson1, e.nameCp1, e.contactPerson2, e.nameCp2, " +
	"e.tag, e.isPublished, e.viewCount, e.authorID, e.createdDate"

// RepositoryImpl is the GORM-backed implementation of Repository.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository creates a Repository backed by the given *gorm.DB.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) base(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Table("ms_event e")
}

func (r *RepositoryImpl) List(ctx context.Context, f event_dto.Filter) ([]event_model.Event, int64, error) {
	q := r.base(ctx)
	if f.PublishedOnly {
		q = q.Where("e.isPublished = 1")
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("(e.eventTitle LIKE ? OR e.eventDivision LIKE ?)", like, like)
	}
	if len(f.Divisions) > 0 {
		q = q.Where("e.eventDivision IN ?", f.Divisions)
	}
	if len(f.Years) > 0 {
		// Filter by year of startDate
		q = q.Where("YEAR(e.startDate) IN ?", f.Years)
	}
	if len(f.Statuses) > 0 {
		now := time.Now()
		var conditions []string
		var args []interface{}
		for _, s := range f.Statuses {
			switch s {
			case "upcoming":
				conditions = append(conditions, "(e.startDate IS NOT NULL AND e.startDate > ?)")
				args = append(args, now)
			case "ongoing":
				conditions = append(conditions, "(e.startDate IS NOT NULL AND e.endDate IS NOT NULL AND e.startDate <= ? AND e.endDate >= ?)")
				args = append(args, now, now)
			case "past":
				conditions = append(conditions, "(e.startDate IS NULL OR e.endDate IS NULL OR e.endDate < ?)")
				args = append(args, now)
			}
		}
		if len(conditions) > 0 {
			q = q.Where("("+strings.Join(conditions, " OR ")+")", args...)
		}
	}

	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Build ORDER BY
	order := buildOrder(f.SortBy, f.SortOrder)
	var out []event_model.Event
	err := q.Select(selectCols).Order(order).Limit(f.Limit).Offset(f.Offset).Find(&out).Error
	return out, total, err
}

// buildOrder converts sort params to a safe SQL ORDER BY expression.
func buildOrder(sortBy, sortOrder string) string {
	allowed := map[string]string{
		"title":       "e.eventTitle",
		"createdDate": "e.createdDate",
		"startDate":   "e.startDate",
		"newest":      "COALESCE(e.startDate, e.createdDate)",
	}
	col, ok := allowed[sortBy]
	if !ok {
		col = "COALESCE(e.startDate, e.createdDate)" // default: newest first
	}
	dir := "DESC"
	if strings.ToUpper(sortOrder) == "ASC" {
		dir = "ASC"
	}
	return fmt.Sprintf("%s %s", col, dir)
}

func (r *RepositoryImpl) findOne(ctx context.Context, where string, arg interface{}) (event_model.Event, error) {
	var e event_model.Event
	err := r.base(ctx).Select(selectCols).Where(where, arg).Take(&e).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return event_model.Event{}, ErrNotFound
	}
	return e, err
}

func (r *RepositoryImpl) FindByID(ctx context.Context, id int64) (event_model.Event, error) {
	return r.findOne(ctx, "e.eventID = ?", id)
}

func (r *RepositoryImpl) FindBySlug(ctx context.Context, slug string) (event_model.Event, error) {
	return r.findOne(ctx, "e.eventSlug = ?", slug)
}

func (r *RepositoryImpl) SlugExists(ctx context.Context, slug string, exceptID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("ms_event").
		Where("eventSlug = ? AND eventID <> ?", slug, exceptID).Count(&count).Error
	return count > 0, err
}

func (r *RepositoryImpl) Create(ctx context.Context, e event_model.Event, authorID int64) (int64, error) {
	values := buildValues(e, authorID)
	values["authorID"] = authorID
	values["createdDate"] = time.Now()
	values["createdBy"] = authorID

	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("ms_event").Create(values).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) Update(ctx context.Context, id int64, e event_model.Event, updatedBy int64) error {
	values := buildValues(e, updatedBy)
	values["updatedDate"] = time.Now()
	values["updatedBy"] = updatedBy
	return r.db.WithContext(ctx).Table("ms_event").Where("eventID = ?", id).Updates(values).Error
}

func (r *RepositoryImpl) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Exec("DELETE FROM ms_event WHERE eventID = ?", id).Error
}

func (r *RepositoryImpl) IncrementViewCount(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Exec("UPDATE ms_event SET viewCount = viewCount + 1 WHERE eventID = ?", id).Error
}

// buildValues maps event model fields to a column-value map for GORM writes.
func buildValues(e event_model.Event, _ int64) map[string]interface{} {
	return map[string]interface{}{
		"eventTitle":       e.EventTitle,
		"eventSlug":        e.EventSlug,
		"eventDivision":    e.EventDivision,
		"eventContent":     e.EventContent,
		"eventImage":       e.EventImage,
		"startDate":        e.StartDate,
		"endDate":          e.EndDate,
		"closeRegistDate":  e.CloseRegistDate,
		"location":         e.Location,
		"place":            e.Place,
		"locationLink":     e.LocationLink,
		"registrationLink": e.RegistrationLink,
		"documentLink":     e.DocumentLink,
		"presentationLink": e.PresentationLink,
		"contactPerson1":   e.ContactPerson1,
		"nameCp1":          e.NameCp1,
		"contactPerson2":   e.ContactPerson2,
		"nameCp2":          e.NameCp2,
		"tag":              e.Tag,
		"isPublished":      e.IsPublished,
	}
}
