package shortlinkrequest_repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"fsldk-api/modules/shortlink/shortlinkrequest_dto"
	"fsldk-api/modules/shortlink/shortlinkrequest_model"

	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

const selectCols = "sr.shortLinkRequestID, sr.requesterName, sr.requesterEmail, sr.requesterWhatsapp, " +
	"sr.destinationURL, sr.requestedKey, sr.note, sr.status, sr.shortLinkID, " +
	"COALESCE(sl.shortKey, '') AS shortKey, sr.rejectionReason, sr.reviewedBy, sr.reviewedVia, " +
	"COALESCE(u.fullName, '') AS reviewerName, sr.reviewedDate, sr.createdDate"

const joins = "LEFT JOIN ms_shortlink sl ON sl.shortLinkID = sr.shortLinkID " +
	"LEFT JOIN ms_user u ON u.userID = sr.reviewedBy"

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

// isDuplicateKeyErr mendeteksi MySQL error 1062 (Duplicate entry) — dipakai
// ApproveTx membedakan collision UNIQUE(shortKey) dari error lain.
func isDuplicateKeyErr(err error) bool {
	var mysqlErr *mysqldriver.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

func (r *RepositoryImpl) FindByID(ctx context.Context, id int64) (shortlinkrequest_model.ShortLinkRequest, error) {
	var sr shortlinkrequest_model.ShortLinkRequest
	err := r.db.WithContext(ctx).Table("ms_shortlink_request sr").
		Select(selectCols).
		Joins(joins).
		Where("sr.shortLinkRequestID = ?", id).
		Take(&sr).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return shortlinkrequest_model.ShortLinkRequest{}, ErrNotFound
	}
	return sr, err
}

func (r *RepositoryImpl) FindPendingByIDs(ctx context.Context, ids []int64) ([]shortlinkrequest_model.ShortLinkRequest, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var out []shortlinkrequest_model.ShortLinkRequest
	err := r.db.WithContext(ctx).Table("ms_shortlink_request sr").
		Select(selectCols).
		Joins(joins).
		Where("sr.shortLinkRequestID IN ? AND sr.status = ?", ids, shortlinkrequest_model.StatusPending).
		Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) List(ctx context.Context, f shortlinkrequest_dto.ListFilter) ([]shortlinkrequest_model.ShortLinkRequest, int64, error) {
	base := r.db.WithContext(ctx).Table("ms_shortlink_request sr").Joins(joins)
	if f.Status != "" {
		base = base.Where("sr.status = ?", f.Status)
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		base = base.Where("(sr.requesterName LIKE ? OR sr.requesterEmail LIKE ? OR sr.destinationURL LIKE ?)", like, like, like)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []shortlinkrequest_model.ShortLinkRequest
	err := base.Select(selectCols).Order(f.OrderBy).Limit(f.Limit).Offset(f.Offset).Find(&out).Error
	return out, total, err
}

func (r *RepositoryImpl) Create(ctx context.Context, req shortlinkrequest_dto.SubmitRequest) (int64, error) {
	values := map[string]interface{}{
		"requesterName":     req.RequesterName,
		"requesterEmail":    req.RequesterEmail,
		"requesterWhatsapp": req.RequesterWhatsapp,
		"destinationURL":    req.DestinationURL,
		"requestedKey":      nullableString(req.RequestedKey),
		"note":              nullableString(req.Note),
		"status":            shortlinkrequest_model.StatusPending,
		"createdDate":       time.Now(),
	}
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("ms_shortlink_request").Create(values).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) ApproveTx(ctx context.Context, requestID int64, destinationURL, shortKey string, reviewerID *int64, reviewedVia string) (int64, error) {
	var newShortLinkID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var createdBy sql.NullInt64
		if reviewerID != nil {
			createdBy = sql.NullInt64{Int64: *reviewerID, Valid: true}
		}
		if err := tx.Table("ms_shortlink").Create(map[string]interface{}{
			"destinationURL": destinationURL,
			"shortKey":       shortKey,
			"visitCount":     0,
			"createdBy":      createdBy,
			"createdDate":    time.Now(),
		}).Error; err != nil {
			if isDuplicateKeyErr(err) {
				return ErrKeyCollision
			}
			return err
		}
		if err := tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newShortLinkID).Error; err != nil {
			return err
		}

		result := tx.Table("ms_shortlink_request").
			Where("shortLinkRequestID = ? AND status = ?", requestID, shortlinkrequest_model.StatusPending).
			Updates(map[string]interface{}{
				"status":       shortlinkrequest_model.StatusApproved,
				"shortLinkID":  newShortLinkID,
				"reviewedBy":   reviewerID,
				"reviewedVia":  reviewedVia,
				"reviewedDate": time.Now(),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			// Kalah race (§1a.5/§6) — rollback juga insert ms_shortlink di atas,
			// tidak ada baris shortlink nyasar untuk request yang gagal diproses.
			return ErrAlreadyProcessed
		}
		return nil
	})
	return newShortLinkID, err
}

func (r *RepositoryImpl) UpdateStatus(ctx context.Context, requestID int64, status string, reviewerID *int64, reviewedVia string, rejectionReason *string) error {
	result := r.db.WithContext(ctx).Table("ms_shortlink_request").
		Where("shortLinkRequestID = ? AND status = ?", requestID, shortlinkrequest_model.StatusPending).
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
