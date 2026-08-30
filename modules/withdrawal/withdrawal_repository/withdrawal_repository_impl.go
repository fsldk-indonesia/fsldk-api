package withdrawal_repository

import (
	"context"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"fsldk-api/constants"
	"fsldk-api/modules/withdrawal/withdrawal_dto"
	"fsldk-api/modules/withdrawal/withdrawal_model"
)

const mysqlDuplicateEntryErrNo = 1062

// nonFinalStatuses adalah status tr_withdrawal yang masih dianggap
// "in-flight" — dana masih direservasi, belum keluar permanen dan belum
// dilepas kembali (reuse daftar yang sama dengan wallet_repository).
var nonFinalStatuses = []string{
	constants.WithdrawalStatusRequested, constants.WithdrawalStatusSecurityCheck,
	constants.WithdrawalStatusPendingApproval, constants.WithdrawalStatusApproved,
	constants.WithdrawalStatusProcessing,
}

const withdrawalSelectCols = "w.withdrawalID, w.withdrawalRef, w.campaignID, c.title AS campaignTitle, " +
	"w.requestedByUserID, w.amount, w.fee, w.netAmount, w.beneficiaryBankCode, w.beneficiaryAccountNumber, " +
	"w.beneficiaryAccountHolder, w.status, w.securityVerifiedDate, w.securityVerifiedMethod, " +
	"w.approvedByUserID, w.approvedDate, w.rejectionReason, " +
	"w.idempotencyKey, w.gatewayStatusID, w.gatewayResponseJSON, w.executedDate, w.completedDate, " +
	"w.createdDate, w.updatedDate"

// RepositoryImpl adalah implementasi Repository berbasis GORM.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) baseQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Table(constants.TableWithdrawal + " w").
		Joins("JOIN " + constants.TableCampaign + " c ON c.campaignID = w.campaignID")
}

func (r *RepositoryImpl) Create(tx *gorm.DB, p withdrawal_model.CreateParams) (int64, error) {
	values := map[string]interface{}{
		"withdrawalRef":            p.WithdrawalRef,
		"campaignID":               p.CampaignID,
		"requestedByUserID":        p.RequestedByUserID,
		"amount":                   p.Amount,
		"fee":                      p.Fee,
		"netAmount":                p.NetAmount,
		"beneficiaryBankCode":      p.BeneficiaryBankCode,
		"beneficiaryAccountNumber": p.BeneficiaryAccountNumber,
		"beneficiaryAccountHolder": p.BeneficiaryAccountHolder,
		"status":                   constants.WithdrawalStatusSecurityCheck,
		"idempotencyKey":           p.IdempotencyKey,
		"createdDate":              time.Now(),
	}
	if err := tx.Table(constants.TableWithdrawal).Create(values).Error; err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlDuplicateEntryErrNo {
			return 0, ErrDuplicateIdempotencyKey
		}
		return 0, err
	}
	var newID int64
	if err := tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error; err != nil {
		return 0, err
	}
	return newID, nil
}

func (r *RepositoryImpl) findOne(ctx context.Context, where string, arg interface{}) (withdrawal_model.Withdrawal, error) {
	var w withdrawal_model.Withdrawal
	err := r.baseQuery(ctx).Select(withdrawalSelectCols).Where(where, arg).Take(&w).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return withdrawal_model.Withdrawal{}, ErrNotFound
	}
	return w, err
}

func (r *RepositoryImpl) FindByID(ctx context.Context, id int64) (withdrawal_model.Withdrawal, error) {
	return r.findOne(ctx, "w.withdrawalID = ?", id)
}

func (r *RepositoryImpl) FindByIdempotencyKey(ctx context.Context, key string) (withdrawal_model.Withdrawal, error) {
	return r.findOne(ctx, "w.idempotencyKey = ?", key)
}

func (r *RepositoryImpl) CountNonFinalByCampaign(ctx context.Context, campaignID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Table(constants.TableWithdrawal).
		Where("campaignID = ? AND status IN ?", campaignID, nonFinalStatuses).
		Count(&count).Error
	return count, err
}

// CountNonFinalByCampaignForUpdate mengunci baris withdrawal non-final milik
// campaign ini (gap lock InnoDB via idx_withdrawal_campaign_status) sehingga
// request konkuren untuk campaign yang sama antre, bukan sama-sama lolos
// pre-check lalu sama-sama insert (TOCTOU).
func (r *RepositoryImpl) CountNonFinalByCampaignForUpdate(tx *gorm.DB, campaignID int64) (int64, error) {
	var count int64
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Table(constants.TableWithdrawal).
		Where("campaignID = ? AND status IN ?", campaignID, nonFinalStatuses).
		Count(&count).Error
	return count, err
}

func (r *RepositoryImpl) List(ctx context.Context, f withdrawal_dto.ListFilter) ([]withdrawal_model.Withdrawal, int64, error) {
	q := r.baseQuery(ctx)
	if f.CampaignID > 0 {
		q = q.Where("w.campaignID = ?", f.CampaignID)
	}
	if f.RequestedByUserID != nil {
		q = q.Where("w.requestedByUserID = ?", *f.RequestedByUserID)
	}
	if f.Status != "" {
		q = q.Where("w.status = ?", f.Status)
	}

	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []withdrawal_model.Withdrawal
	err := q.Select(withdrawalSelectCols).Order(f.OrderBy).Limit(f.Limit).Offset(f.Offset).Find(&out).Error
	return out, total, err
}

func (r *RepositoryImpl) UpdateStatus(tx *gorm.DB, id int64, status string, p withdrawal_model.StatusUpdateParams) error {
	values := map[string]interface{}{
		"status":      status,
		"updatedDate": time.Now(),
	}
	if p.ApprovedByUserID != nil {
		values["approvedByUserID"] = *p.ApprovedByUserID
		values["approvedDate"] = time.Now()
	}
	if p.RejectionReason != nil {
		values["rejectionReason"] = *p.RejectionReason
	}
	if p.GatewayStatusID != nil {
		values["gatewayStatusID"] = *p.GatewayStatusID
	}
	if p.GatewayResponseJSON != nil {
		values["gatewayResponseJSON"] = *p.GatewayResponseJSON
	}
	if p.SecurityVerifiedMethod != nil {
		values["securityVerifiedMethod"] = *p.SecurityVerifiedMethod
	}
	if p.SetExecutedNow {
		values["executedDate"] = time.Now()
	}
	if p.SetCompletedNow {
		values["completedDate"] = time.Now()
	}
	if p.SetSecurityVerifiedNow {
		values["securityVerifiedDate"] = time.Now()
	}
	return tx.Table(constants.TableWithdrawal).Where("withdrawalID = ?", id).Updates(values).Error
}

func (r *RepositoryImpl) FindByRefForUpdate(tx *gorm.DB, withdrawalRef string) (withdrawal_model.Withdrawal, error) {
	var w withdrawal_model.Withdrawal
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Table(constants.TableWithdrawal).
		Where("withdrawalRef = ?", withdrawalRef).
		Take(&w).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return withdrawal_model.Withdrawal{}, ErrNotFound
	}
	return w, err
}

func (r *RepositoryImpl) FindStaleProcessing(ctx context.Context, olderThan time.Time) ([]withdrawal_model.Withdrawal, error) {
	var out []withdrawal_model.Withdrawal
	err := r.baseQuery(ctx).Select(withdrawalSelectCols).
		Where("w.status = ? AND w.executedDate < ?", constants.WithdrawalStatusProcessing, olderThan).
		Order("w.executedDate ASC").
		Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) CountSuccessByCampaign(ctx context.Context, campaignID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Table(constants.TableWithdrawal).
		Where("campaignID = ? AND status = ?", campaignID, constants.WithdrawalStatusSuccess).
		Count(&count).Error
	return count, err
}

const otpChallengeTable = "tr_otp_challenge"

func (r *RepositoryImpl) CreateOtpChallenge(ctx context.Context, p withdrawal_model.OtpChallengeParams) (int64, error) {
	values := map[string]interface{}{
		"withdrawalID": p.WithdrawalID,
		"userID":       p.UserID,
		"codeHash":     p.CodeHash,
		"channel":      p.Channel,
		"expiredDate":  p.ExpiredDate,
		"createdDate":  time.Now(),
	}
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table(otpChallengeTable).Create(values).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) FindActiveOtpChallenge(ctx context.Context, withdrawalID int64) (withdrawal_model.OtpChallenge, error) {
	var c withdrawal_model.OtpChallenge
	err := r.db.WithContext(ctx).Table(otpChallengeTable).
		Where("withdrawalID = ? AND verifiedDate IS NULL AND expiredDate > ?", withdrawalID, time.Now()).
		Order("challengeID DESC").
		Take(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return withdrawal_model.OtpChallenge{}, ErrNotFound
	}
	return c, err
}

func (r *RepositoryImpl) IncrementOtpAttempt(ctx context.Context, challengeID int64) error {
	return r.db.WithContext(ctx).Table(otpChallengeTable).
		Where("challengeID = ?", challengeID).
		UpdateColumn("attemptCount", gorm.Expr("attemptCount + 1")).Error
}

func (r *RepositoryImpl) MarkOtpVerified(ctx context.Context, challengeID int64) error {
	return r.db.WithContext(ctx).Table(otpChallengeTable).
		Where("challengeID = ?", challengeID).
		Update("verifiedDate", time.Now()).Error
}

func (r *RepositoryImpl) CountOtpChallengesByWithdrawal(ctx context.Context, withdrawalID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Table(otpChallengeTable).
		Where("withdrawalID = ?", withdrawalID).
		Count(&count).Error
	return count, err
}
