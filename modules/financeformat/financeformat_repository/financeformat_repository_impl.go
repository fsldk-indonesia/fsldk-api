package financeformat_repository

import (
	"context"
	"errors"
	"time"

	"fsldk-api/modules/financeformat/financeformat_dto"
	"fsldk-api/modules/financeformat/financeformat_model"

	"gorm.io/gorm"
)

const selectCols = "f.financeFormatID, f.fileName, f.fileURL, f.formatTypeID, " +
	"t.formatTypeName, f.isActive, f.createdDate"

// RepositoryImpl is the GORM-based Repository implementation.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository creates the Repository implementation.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) baseQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Table("ms_finance_format f").
		Joins("JOIN lk_finance_format_type t ON t.formatTypeID = f.formatTypeID")
}

func (r *RepositoryImpl) List(ctx context.Context, f financeformat_dto.Filter) ([]financeformat_model.FinanceFormat, int64, error) {
	q := r.baseQuery(ctx)
	if f.ActiveOnly {
		q = q.Where("f.isActive = 1")
	}
	if f.Search != "" {
		q = q.Where("f.fileName LIKE ?", "%"+f.Search+"%")
	}
	if f.FormatTypeID > 0 {
		q = q.Where("f.formatTypeID = ?", f.FormatTypeID)
	}
	if f.DateFrom != "" {
		q = q.Where("DATE(f.createdDate) >= ?", f.DateFrom)
	}
	if f.DateTo != "" {
		q = q.Where("DATE(f.createdDate) <= ?", f.DateTo)
	}

	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	q = q.Select(selectCols).Order(f.OrderBy)
	if f.Limit > 0 {
		q = q.Limit(f.Limit).Offset(f.Offset)
	}

	var out []financeformat_model.FinanceFormat
	err := q.Find(&out).Error
	return out, total, err
}

func (r *RepositoryImpl) FindByID(ctx context.Context, id int64) (financeformat_model.FinanceFormat, error) {
	var m financeformat_model.FinanceFormat
	err := r.baseQuery(ctx).Select(selectCols).Where("f.financeFormatID = ?", id).Take(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return financeformat_model.FinanceFormat{}, ErrNotFound
	}
	return m, err
}

func (r *RepositoryImpl) Create(ctx context.Context, m financeformat_model.FinanceFormat, actorID int64) (int64, error) {
	values := map[string]interface{}{
		"fileName":     m.FileName,
		"fileURL":      m.FileURL,
		"formatTypeID": m.FormatTypeID,
		"isActive":     true,
		"createdDate":  time.Now(),
		"createdBy":    actorID,
	}
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("ms_finance_format").Create(values).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) Update(ctx context.Context, id int64, m financeformat_model.FinanceFormat, actorID int64) error {
	return r.db.WithContext(ctx).Table("ms_finance_format").Where("financeFormatID = ?", id).Updates(map[string]interface{}{
		"fileName":     m.FileName,
		"fileURL":      m.FileURL,
		"formatTypeID": m.FormatTypeID,
		"updatedDate":  time.Now(),
		"updatedBy":    actorID,
	}).Error
}

func (r *RepositoryImpl) SetActive(ctx context.Context, id int64, isActive bool, actorID int64) error {
	return r.db.WithContext(ctx).Table("ms_finance_format").Where("financeFormatID = ?", id).Updates(map[string]interface{}{
		"isActive":    isActive,
		"updatedDate": time.Now(),
		"updatedBy":   actorID,
	}).Error
}

func (r *RepositoryImpl) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Exec("DELETE FROM ms_finance_format WHERE financeFormatID = ?", id).Error
}

func (r *RepositoryImpl) ListFormatTypes(ctx context.Context) ([]financeformat_model.FormatType, error) {
	var out []financeformat_model.FormatType
	err := r.db.WithContext(ctx).Table("lk_finance_format_type").Order("sortOrder").Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) FormatTypeExists(ctx context.Context, id int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("lk_finance_format_type").Where("formatTypeID = ?", id).Count(&count).Error
	return count > 0, err
}
