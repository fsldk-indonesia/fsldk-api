package shortlink_repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"fsldk-api/modules/shortlink/shortlink_dto"
	"fsldk-api/modules/shortlink/shortlink_model"

	"gorm.io/gorm"
)

const selectCols = "s.shortLinkID, s.shortKey, s.destinationURL, s.visitCount, s.createdBy, " +
	"u.fullName AS authorName, s.createdDate, s.updatedBy, s.updatedDate"

const joinAuthor = "LEFT JOIN ms_user u ON u.userID = s.createdBy"

// RepositoryImpl adalah implementasi Repository berbasis GORM.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) findOne(ctx context.Context, where string, arg interface{}) (shortlink_model.ShortLink, error) {
	var s shortlink_model.ShortLink
	err := r.db.WithContext(ctx).Table("ms_shortlink s").
		Select(selectCols).
		Joins(joinAuthor).
		Where(where, arg).
		Take(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return shortlink_model.ShortLink{}, ErrNotFound
	}
	return s, err
}

func (r *RepositoryImpl) FindByID(ctx context.Context, id int64) (shortlink_model.ShortLink, error) {
	return r.findOne(ctx, "s.shortLinkID = ?", id)
}

func (r *RepositoryImpl) FindByKey(ctx context.Context, key string) (shortlink_model.ShortLink, error) {
	return r.findOne(ctx, "s.shortKey = ?", key)
}

func (r *RepositoryImpl) ExistsByKey(ctx context.Context, key string, exceptID int64) (bool, error) {
	q := r.db.WithContext(ctx).Table("ms_shortlink").Where("shortKey = ?", key)
	if exceptID > 0 {
		q = q.Where("shortLinkID <> ?", exceptID)
	}
	var count int64
	err := q.Count(&count).Error
	return count > 0, err
}

func (r *RepositoryImpl) List(ctx context.Context, f shortlink_dto.ListFilter) ([]shortlink_model.ShortLink, int64, error) {
	base := r.db.WithContext(ctx).Table("ms_shortlink s").Joins(joinAuthor)
	if f.Search != "" {
		like := "%" + f.Search + "%"
		base = base.Where("(s.shortKey LIKE ? OR s.destinationURL LIKE ?)", like, like)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []shortlink_model.ShortLink
	err := base.Select(selectCols).Order(f.OrderBy).Limit(f.Limit).Offset(f.Offset).Find(&out).Error
	return out, total, err
}

func (r *RepositoryImpl) Create(ctx context.Context, shortKey, destinationURL string, createdBy int64) (int64, error) {
	values := map[string]interface{}{
		"shortKey":       shortKey,
		"destinationURL": destinationURL,
		"visitCount":     0,
		"createdBy":      sql.NullInt64{Int64: createdBy, Valid: createdBy > 0},
		"createdDate":    time.Now(),
	}
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("ms_shortlink").Create(values).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) Update(ctx context.Context, id int64, shortKey, destinationURL string, updatedBy int64) error {
	return r.db.WithContext(ctx).Table("ms_shortlink").Where("shortLinkID = ?", id).Updates(map[string]interface{}{
		"shortKey":       shortKey,
		"destinationURL": destinationURL,
		"updatedDate":    time.Now(),
		"updatedBy":      updatedBy,
	}).Error
}

func (r *RepositoryImpl) IncrementVisit(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Exec("UPDATE ms_shortlink SET visitCount = visitCount + 1 WHERE shortLinkID = ?", id).Error
}

func (r *RepositoryImpl) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Exec("DELETE FROM ms_shortlink WHERE shortLinkID = ?", id).Error
}
