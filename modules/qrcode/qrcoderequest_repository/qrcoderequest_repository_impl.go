package qrcoderequest_repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"fsldk-api/modules/qrcode/qrcode_model"
	"fsldk-api/modules/qrcode/qrcoderequest_dto"
	"fsldk-api/modules/qrcode/qrcoderequest_model"

	"gorm.io/gorm"
)

const selectCols = "qr.qrCodeRequestID, qr.requesterName, qr.requesterEmail, qr.requesterWhatsapp, " +
	"qr.destinationURL, qr.foregroundColor, qr.backgroundColor, qr.centerIconURL, qr.centerIconKey, qr.captionText, " +
	"qr.note, qr.status, qr.qrCodeID, qr.rejectionReason, qr.reviewedBy, " +
	"qr.reviewedVia, COALESCE(u.fullName, '') AS reviewerName, qr.reviewedDate, qr.createdDate"

const joins = "LEFT JOIN ms_user u ON u.userID = qr.reviewedBy"

// RepositoryImpl adalah implementasi Repository berbasis GORM.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (r *RepositoryImpl) FindByID(ctx context.Context, id int64) (qrcoderequest_model.QRCodeRequest, error) {
	var qr qrcoderequest_model.QRCodeRequest
	err := r.db.WithContext(ctx).Table("ms_qrcode_request qr").
		Select(selectCols).
		Joins(joins).
		Where("qr.qrCodeRequestID = ?", id).
		Take(&qr).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return qrcoderequest_model.QRCodeRequest{}, ErrNotFound
	}
	return qr, err
}

func (r *RepositoryImpl) FindPendingByIDs(ctx context.Context, ids []int64) ([]qrcoderequest_model.QRCodeRequest, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var out []qrcoderequest_model.QRCodeRequest
	err := r.db.WithContext(ctx).Table("ms_qrcode_request qr").
		Select(selectCols).
		Joins(joins).
		Where("qr.qrCodeRequestID IN ? AND qr.status = ?", ids, qrcoderequest_model.StatusPending).
		Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) List(ctx context.Context, f qrcoderequest_dto.ListFilter) ([]qrcoderequest_model.QRCodeRequest, int64, error) {
	base := r.db.WithContext(ctx).Table("ms_qrcode_request qr").Joins(joins)
	if f.Status != "" {
		base = base.Where("qr.status = ?", f.Status)
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		base = base.Where("(qr.requesterName LIKE ? OR qr.requesterEmail LIKE ? OR qr.destinationURL LIKE ?)", like, like, like)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []qrcoderequest_model.QRCodeRequest
	err := base.Select(selectCols).Order(f.OrderBy).Limit(f.Limit).Offset(f.Offset).Find(&out).Error
	return out, total, err
}

func (r *RepositoryImpl) Create(ctx context.Context, req qrcoderequest_dto.SubmitRequest) (int64, error) {
	values := map[string]interface{}{
		"requesterName":     req.RequesterName,
		"requesterEmail":    req.RequesterEmail,
		"requesterWhatsapp": req.RequesterWhatsapp,
		"destinationURL":    req.DestinationURL,
		"foregroundColor":   req.ForegroundColor,
		"backgroundColor":   req.BackgroundColor,
		"centerIconURL":     nullableString(req.CenterIconURL),
		"centerIconKey":     nullableString(req.CenterIconKey),
		"captionText":       nullableString(req.CaptionText),
		"note":              nullableString(req.Note),
		"status":            qrcoderequest_model.StatusPending,
		"createdDate":       time.Now(),
	}
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("ms_qrcode_request").Create(values).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func iconOrNil(ns sql.NullString) interface{} {
	if ns.Valid && ns.String != "" {
		return ns.String
	}
	return nil
}

func (r *RepositoryImpl) ApproveTx(ctx context.Context, requestID int64, qr qrcode_model.QRCode, reviewerID *int64, reviewedVia string) (int64, error) {
	var newQRCodeID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var createdBy sql.NullInt64
		if reviewerID != nil {
			createdBy = sql.NullInt64{Int64: *reviewerID, Valid: true}
		}
		if err := tx.Table("ms_qrcode").Create(map[string]interface{}{
			"destinationURL":  qr.DestinationURL,
			"foregroundColor": qr.ForegroundColor,
			"backgroundColor": qr.BackgroundColor,
			"centerIconURL":   iconOrNil(qr.CenterIconURL),
			"centerIconKey":   iconOrNil(qr.CenterIconKey),
			"captionText":     iconOrNil(qr.CaptionText),
			"createdBy":       createdBy,
			"createdDate":     time.Now(),
		}).Error; err != nil {
			return err
		}
		if err := tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newQRCodeID).Error; err != nil {
			return err
		}

		result := tx.Table("ms_qrcode_request").
			Where("qrCodeRequestID = ? AND status = ?", requestID, qrcoderequest_model.StatusPending).
			Updates(map[string]interface{}{
				"status":       qrcoderequest_model.StatusApproved,
				"qrCodeID":     newQRCodeID,
				"reviewedBy":   reviewerID,
				"reviewedVia":  reviewedVia,
				"reviewedDate": time.Now(),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			// Kalah race — rollback juga insert ms_qrcode di atas.
			return ErrAlreadyProcessed
		}
		return nil
	})
	return newQRCodeID, err
}

func (r *RepositoryImpl) UpdateStatus(ctx context.Context, requestID int64, status string, reviewerID *int64, reviewedVia string, rejectionReason *string) error {
	result := r.db.WithContext(ctx).Table("ms_qrcode_request").
		Where("qrCodeRequestID = ? AND status = ?", requestID, qrcoderequest_model.StatusPending).
		Updates(map[string]interface{}{
			"status":          status,
			"rejectionReason": rejectionReason,
			"reviewedBy":      reviewerID,
			"reviewedVia":     reviewedVia,
			"reviewedDate":    time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrAlreadyProcessed
	}
	return nil
}
