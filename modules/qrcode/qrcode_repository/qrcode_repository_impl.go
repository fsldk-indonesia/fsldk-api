package qrcode_repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"fsldk-api/modules/qrcode/qrcode_dto"
	"fsldk-api/modules/qrcode/qrcode_model"

	"gorm.io/gorm"
)

const selectCols = "q.qrCodeID, q.label, q.destinationURL, q.foregroundColor, q.backgroundColor, " +
	"q.centerIconURL, q.centerIconKey, q.captionText, q.createdBy, u.fullName AS authorName, " +
	"q.createdDate, q.updatedBy, q.updatedDate"

const joinAuthor = "LEFT JOIN ms_user u ON u.userID = q.createdBy"

// RepositoryImpl adalah implementasi Repository berbasis GORM.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func nullableString(ns sql.NullString) interface{} {
	if ns.Valid && ns.String != "" {
		return ns.String
	}
	return nil
}

func (r *RepositoryImpl) FindByID(ctx context.Context, id int64) (qrcode_model.QRCode, error) {
	var q qrcode_model.QRCode
	err := r.db.WithContext(ctx).Table("ms_qrcode q").
		Select(selectCols).
		Joins(joinAuthor).
		Where("q.qrCodeID = ?", id).
		Take(&q).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return qrcode_model.QRCode{}, ErrNotFound
	}
	return q, err
}

func (r *RepositoryImpl) List(ctx context.Context, f qrcode_dto.ListFilter) ([]qrcode_model.QRCode, int64, error) {
	base := r.db.WithContext(ctx).Table("ms_qrcode q").Joins(joinAuthor)
	if f.Search != "" {
		like := "%" + f.Search + "%"
		base = base.Where("(q.label LIKE ? OR q.destinationURL LIKE ? OR q.captionText LIKE ?)", like, like, like)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []qrcode_model.QRCode
	err := base.Select(selectCols).Order(f.OrderBy).Limit(f.Limit).Offset(f.Offset).Find(&out).Error
	return out, total, err
}

func (r *RepositoryImpl) Create(ctx context.Context, m qrcode_model.QRCode) (int64, error) {
	values := map[string]interface{}{
		"label":           nullableString(m.Label),
		"destinationURL":  m.DestinationURL,
		"foregroundColor": m.ForegroundColor,
		"backgroundColor": m.BackgroundColor,
		"centerIconURL":   nullableString(m.CenterIconURL),
		"centerIconKey":   nullableString(m.CenterIconKey),
		"captionText":     nullableString(m.CaptionText),
		"createdBy":       m.CreatedBy,
		"createdDate":     time.Now(),
	}
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("ms_qrcode").Create(values).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) Update(ctx context.Context, id int64, m qrcode_model.QRCode) error {
	return r.db.WithContext(ctx).Table("ms_qrcode").Where("qrCodeID = ?", id).Updates(map[string]interface{}{
		"label":           nullableString(m.Label),
		"destinationURL":  m.DestinationURL,
		"foregroundColor": m.ForegroundColor,
		"backgroundColor": m.BackgroundColor,
		"centerIconURL":   nullableString(m.CenterIconURL),
		"centerIconKey":   nullableString(m.CenterIconKey),
		"captionText":     nullableString(m.CaptionText),
		"updatedDate":     time.Now(),
		"updatedBy":       m.UpdatedBy,
	}).Error
}

func (r *RepositoryImpl) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Exec("DELETE FROM ms_qrcode WHERE qrCodeID = ?", id).Error
}
