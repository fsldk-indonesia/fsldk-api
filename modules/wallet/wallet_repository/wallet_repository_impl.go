package wallet_repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"fsldk-api/constants"
	"fsldk-api/modules/wallet/wallet_dto"
	"fsldk-api/modules/wallet/wallet_model"
)

const mysqlDuplicateEntryErrNo = 1062

// applyDirection menghitung balance baru dari balance sebelumnya + satu
// ledger entry — fungsi murni, diuji langsung tanpa DB (lihat
// balance_test.go).
func applyDirection(prevBalance, amount float64, direction string) float64 {
	switch direction {
	case constants.LedgerDirectionCredit:
		return prevBalance + amount
	case constants.LedgerDirectionDebit:
		return prevBalance - amount
	default:
		return prevBalance
	}
}

// nonFinalWithdrawalStatuses adalah status tr_withdrawal yang masih
// dianggap "in-flight" — dana masih direservasi, belum keluar permanen
// dan belum dilepas kembali.
var nonFinalWithdrawalStatuses = []string{
	"REQUESTED", "SECURITY_CHECK", "PENDING_APPROVAL", "APPROVED", "PROCESSING",
}

type RepositoryImpl struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) InsertLedgerEntry(tx *gorm.DB, p wallet_model.LedgerEntryParams) (int64, float64, error) {
	// Mengunci baris ms_campaign lalu membaca balanceAfter terakhir —
	// menyerialkan seluruh mutasi ledger campaign ini (konkuren donation
	// callback/withdrawal/adjustment harus antre di sini). Aman dipanggil
	// lagi walau caller (mis. wallet_service.ReserveWithdrawal) sudah memanggil
	// LatestBalanceForUpdate lebih dulu di tx yang sama — lock InnoDB
	// reentrant per-transaksi.
	prevBalance, err := r.LatestBalanceForUpdate(tx, p.CampaignID)
	if err != nil {
		return 0, 0, err
	}

	newBalance := applyDirection(prevBalance, p.Amount, p.Direction)

	if err := tx.Table(constants.TableWalletLedger).Create(map[string]interface{}{
		"campaignID":      p.CampaignID,
		"entryType":       p.EntryType,
		"direction":       p.Direction,
		"amount":          p.Amount,
		"balanceAfter":    newBalance,
		"referenceType":   p.ReferenceType,
		"referenceID":     p.ReferenceID,
		"note":            p.Note,
		"createdByUserID": p.CreatedByUserID,
		"createdDate":     time.Now(),
	}).Error; err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlDuplicateEntryErrNo {
			return 0, 0, ErrDuplicateLedgerEntry
		}
		return 0, 0, err
	}

	var newID int64
	if err := tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error; err != nil {
		return 0, 0, err
	}

	// collectedAmountCache adalah snapshot read-optimized untuk listing
	// publik, bukan sumber kebenaran — diperbarui dalam transaksi yang
	// sama setiap kali DONATION_CREDIT dicatat.
	if p.EntryType == constants.LedgerEntryDonationCredit {
		if err := tx.Table(constants.TableCampaign).
			Where("campaignID = ?", p.CampaignID).
			UpdateColumn("collectedAmountCache", gorm.Expr("collectedAmountCache + ?", p.Amount)).Error; err != nil {
			return 0, 0, err
		}
	}

	return newID, newBalance, nil
}

// readLatestBalance membaca balanceAfter baris ledger terakhir campaign
// memakai tx/db yang diberikan, tanpa mengunci apa pun sendiri (caller
// bertanggung jawab mengunci baris campaign lebih dulu bila diperlukan).
func readLatestBalance(db *gorm.DB, campaignID int64) (float64, error) {
	var row struct {
		BalanceAfter sql.NullFloat64 `gorm:"column:balanceAfter"`
	}
	err := db.Table(constants.TableWalletLedger).
		Select("balanceAfter").
		Where("campaignID = ?", campaignID).
		Order("ledgerID DESC").
		Limit(1).
		Find(&row).Error
	if err != nil {
		return 0, err
	}
	if !row.BalanceAfter.Valid {
		return 0, nil
	}
	return row.BalanceAfter.Float64, nil
}

func (r *RepositoryImpl) LatestBalance(ctx context.Context, campaignID int64) (float64, error) {
	return readLatestBalance(r.db.WithContext(ctx), campaignID)
}

func (r *RepositoryImpl) LatestBalanceForUpdate(tx *gorm.DB, campaignID int64) (float64, error) {
	var lockTarget struct {
		CampaignID int64 `gorm:"column:campaignID"`
	}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Table(constants.TableCampaign).
		Select("campaignID").
		Where("campaignID = ?", campaignID).
		Take(&lockTarget).Error; err != nil {
		return 0, err
	}
	return readLatestBalance(tx, campaignID)
}

func (r *RepositoryImpl) SumByEntryType(ctx context.Context, campaignID int64, entryType string) (float64, error) {
	var row struct {
		Total sql.NullFloat64 `gorm:"column:total"`
	}
	err := r.db.WithContext(ctx).Table(constants.TableWalletLedger).
		Select("SUM(amount) AS total").
		Where("campaignID = ? AND entryType = ?", campaignID, entryType).
		Find(&row).Error
	if err != nil {
		return 0, err
	}
	return row.Total.Float64, nil
}

func (r *RepositoryImpl) SumPendingReserve(ctx context.Context, campaignID int64) (float64, error) {
	var row struct {
		Total sql.NullFloat64 `gorm:"column:total"`
	}
	err := r.db.WithContext(ctx).
		Table(constants.TableWalletLedger+" l").
		Select("SUM(CASE WHEN l.entryType = ? THEN l.amount WHEN l.entryType = ? THEN -l.amount ELSE 0 END) AS total",
			constants.LedgerEntryWithdrawalReserve, constants.LedgerEntryWithdrawalRelease).
		Joins("JOIN "+constants.TableWithdrawal+" w ON w.withdrawalID = l.referenceID AND l.referenceType = ?", constants.LedgerReferenceWithdrawal).
		Where("l.campaignID = ? AND w.status IN ?", campaignID, nonFinalWithdrawalStatuses).
		Find(&row).Error
	if err != nil {
		return 0, err
	}
	return row.Total.Float64, nil
}

func (r *RepositoryImpl) SumWithdrawnSuccess(ctx context.Context, campaignID int64) (float64, error) {
	var row struct {
		Total sql.NullFloat64 `gorm:"column:total"`
	}
	err := r.db.WithContext(ctx).Table(constants.TableWithdrawal).
		Select("SUM(amount) AS total").
		Where("campaignID = ? AND status = 'SUCCESS'", campaignID).
		Find(&row).Error
	if err != nil {
		return 0, err
	}
	return row.Total.Float64, nil
}

func (r *RepositoryImpl) ListLedger(ctx context.Context, campaignID int64, filter wallet_dto.LedgerListFilter) ([]wallet_model.LedgerEntry, int64, error) {
	base := r.db.WithContext(ctx).Table(constants.TableWalletLedger).Where("campaignID = ?", campaignID)
	if filter.EntryType != "" {
		base = base.Where("entryType = ?", filter.EntryType)
	}
	if filter.DateFrom != nil {
		base = base.Where("createdDate >= ?", *filter.DateFrom)
	}
	if filter.DateTo != nil {
		base = base.Where("createdDate <= ?", *filter.DateTo)
	}
	base = base.Session(&gorm.Session{})

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, limit := filter.Page, filter.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var entries []wallet_model.LedgerEntry
	if err := base.Order("ledgerID DESC").Offset((page - 1) * limit).Limit(limit).Find(&entries).Error; err != nil {
		return nil, 0, err
	}
	return entries, total, nil
}
