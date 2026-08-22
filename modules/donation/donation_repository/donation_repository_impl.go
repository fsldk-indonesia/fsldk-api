package donation_repository

import (
	"context"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"fsldk-api/constants"
	"fsldk-api/modules/donation/donation_dto"
	"fsldk-api/modules/donation/donation_model"
)

const mysqlDuplicateEntryErrNo = 1062

const donationSelectCols = "d.donationID, d.publicRef, d.campaignID, c.title AS campaignTitle, c.slug AS campaignSlug, " +
	"d.donorUserID, d.donorName, d.donorEmail, d.donorPhone, d.donorAge, d.donorDomicile, d.donorOccupation, " +
	"d.isAnonymous, d.message, d.amount, d.adminFee, d.totalAmount, d.paymentStatus, d.gateway, " +
	"d.externalTransactionID, d.idempotencyKey, d.qrPayload, d.paymentCode, d.paymentLink, d.gatewayStatusID, " +
	"d.expiredDate, d.createdDate, d.updatedDate"

// RepositoryImpl adalah implementasi Repository berbasis GORM.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) baseQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Table(constants.TableDonation + " d").
		Joins("JOIN " + constants.TableCampaign + " c ON c.campaignID = d.campaignID")
}

func (r *RepositoryImpl) Create(ctx context.Context, p donation_model.CreateParams) (int64, error) {
	values := map[string]interface{}{
		"publicRef":       p.PublicRef,
		"campaignID":      p.CampaignID,
		"donorUserID":     p.DonorUserID,
		"donorName":       p.DonorName,
		"donorEmail":      p.DonorEmail,
		"donorPhone":      p.DonorPhone,
		"donorAge":        p.DonorAge,
		"donorDomicile":   p.DonorDomicile,
		"donorOccupation": p.DonorOccupation,
		"isAnonymous":     p.IsAnonymous,
		"message":         p.Message,
		"amount":          p.Amount,
		"adminFee":        p.AdminFee,
		"totalAmount":     p.TotalAmount,
		"paymentStatus":   constants.DonationStatusPending,
		"gateway":         p.Gateway,
		"idempotencyKey":  p.IdempotencyKey,
		"expiredDate":     p.ExpiredDate,
		"createdDate":     time.Now(),
	}
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table(constants.TableDonation).Create(values).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlDuplicateEntryErrNo {
			return 0, ErrDuplicateIdempotencyKey
		}
		return 0, err
	}
	return newID, nil
}

func (r *RepositoryImpl) findOne(ctx context.Context, where string, arg interface{}) (donation_model.Donation, error) {
	var d donation_model.Donation
	err := r.baseQuery(ctx).Select(donationSelectCols).Where(where, arg).Take(&d).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return donation_model.Donation{}, ErrNotFound
	}
	return d, err
}

func (r *RepositoryImpl) FindByID(ctx context.Context, id int64) (donation_model.Donation, error) {
	return r.findOne(ctx, "d.donationID = ?", id)
}

func (r *RepositoryImpl) FindByPublicRef(ctx context.Context, publicRef string) (donation_model.Donation, error) {
	return r.findOne(ctx, "d.publicRef = ?", publicRef)
}

func (r *RepositoryImpl) FindByIdempotencyKey(ctx context.Context, key string) (donation_model.Donation, error) {
	return r.findOne(ctx, "d.idempotencyKey = ?", key)
}

func (r *RepositoryImpl) List(ctx context.Context, f donation_dto.ListFilter) ([]donation_model.Donation, int64, error) {
	q := r.baseQuery(ctx)
	if f.CampaignID > 0 {
		q = q.Where("d.campaignID = ?", f.CampaignID)
	}
	if f.DonorUserID != nil {
		q = q.Where("d.donorUserID = ?", *f.DonorUserID)
	}
	if f.Status != "" {
		q = q.Where("d.paymentStatus = ?", f.Status)
	}

	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []donation_model.Donation
	err := q.Select(donationSelectCols).Order(f.OrderBy).Limit(f.Limit).Offset(f.Offset).Find(&out).Error
	return out, total, err
}

func (r *RepositoryImpl) UpdateGatewayResult(ctx context.Context, donationID int64, p donation_model.GatewayResultParams) error {
	return r.db.WithContext(ctx).Table(constants.TableDonation).Where("donationID = ?", donationID).Updates(map[string]interface{}{
		"externalTransactionID": p.ExternalTransactionID,
		"qrPayload":             p.QrPayload,
		"paymentCode":           p.PaymentCode,
		"paymentLink":           p.PaymentLink,
		"updatedDate":           time.Now(),
	}).Error
}

func (r *RepositoryImpl) MarkGatewayFailed(ctx context.Context, donationID int64) error {
	return r.db.WithContext(ctx).Table(constants.TableDonation).Where("donationID = ?", donationID).Updates(map[string]interface{}{
		"paymentStatus": constants.DonationStatusFailed,
		"updatedDate":   time.Now(),
	}).Error
}

// FindByExternalTransactionIDForUpdate sengaja tidak melakukan JOIN ke
// ms_campaign (berbeda dari findOne) — callback hanya butuh kolom tr_donation
// sendiri, dan menghindari JOIN di sini mencegah baris ms_campaign ikut
// terkunci tanpa perlu (MySQL mengunci baris dari seluruh tabel yang di-JOIN
// pada SELECT...FOR UPDATE).
func (r *RepositoryImpl) FindByExternalTransactionIDForUpdate(tx *gorm.DB, externalTransactionID string) (donation_model.Donation, error) {
	var d donation_model.Donation
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Table(constants.TableDonation).
		Where("externalTransactionID = ?", externalTransactionID).
		Take(&d).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return donation_model.Donation{}, ErrNotFound
	}
	return d, err
}

func (r *RepositoryImpl) UpdateCallbackStatus(tx *gorm.DB, donationID int64, p donation_model.CallbackUpdateParams) error {
	values := map[string]interface{}{
		"paymentStatus":   p.PaymentStatus,
		"gatewayStatusID": p.GatewayStatusID,
		"updatedDate":     time.Now(),
	}
	if p.TotalAmount != nil {
		values["totalAmount"] = *p.TotalAmount
	}
	if p.AdminFee != nil {
		values["adminFee"] = *p.AdminFee
	}
	return tx.Table(constants.TableDonation).Where("donationID = ?", donationID).Updates(values).Error
}
