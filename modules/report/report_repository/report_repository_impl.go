package report_repository

import (
	"context"
	"database/sql"
	"time"

	"fsldk-api/base/dto"
	"fsldk-api/constants"
	"fsldk-api/modules/report/report_dto"

	"gorm.io/gorm"
)

// RepositoryImpl adalah implementasi Repository berbasis GORM.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) SubmissionRows(ctx context.Context, formID int64, status string, organizationIDs []int64) ([]report_dto.SubmissionRow, error) {
	if len(organizationIDs) == 0 {
		return []report_dto.SubmissionRow{}, nil
	}
	q := r.db.WithContext(ctx).Table("tr_submission s").
		Select("o.organizationName, o.provinceName, o.cityName, s.status, l.levelLabel, s.submittedDate, s.lastUpdatedDate").
		Joins("JOIN ms_organization o ON o.organizationID = s.organizationID").
		Joins("LEFT JOIN tr_levelisasi_result res ON res.organizationID = s.organizationID AND res.isCurrent = 1").
		Joins("LEFT JOIN lk_level l ON l.levelCode = res.levelCode").
		Where("s.formID = ? AND s.organizationID IN ?", formID, organizationIDs)
	if status != "" {
		q = q.Where("s.status = ?", status)
	}
	var out []report_dto.SubmissionRow
	err := q.Order("o.organizationName ASC").Find(&out).Error
	return out, err
}

// ---------- Kantong Amal (Phase 9) — §15 techspec ----------

func (r *RepositoryImpl) balanceAt(ctx context.Context, campaignID int64, cutoff time.Time, inclusive bool) (float64, error) {
	op := "<"
	if inclusive {
		op = "<="
	}
	var row struct {
		Total sql.NullFloat64 `gorm:"column:total"`
	}
	if campaignID > 0 {
		err := r.db.WithContext(ctx).Table(constants.TableWalletLedger).
			Select("balanceAfter AS total").
			Where("campaignID = ? AND createdDate "+op+" ?", campaignID, cutoff).
			Order("ledgerID DESC").Limit(1).Find(&row).Error
		return row.Total.Float64, err
	}
	// Agregat seluruh campaign: balanceAfter terakhir TIAP campaign sebelum
	// cutoff, dijumlahkan. Self-join ke MAX(ledgerID) per campaignID, BUKAN
	// window function (ROW_NUMBER() OVER (...)) — server dev/prod stack ini
	// masih MySQL 5.7 (dikonfirmasi saat integration smoke test), window
	// function baru ada sejak MySQL 8.0.
	err := r.db.WithContext(ctx).Raw(`
		SELECT COALESCE(SUM(l.balanceAfter), 0) AS total
		FROM `+constants.TableWalletLedger+` l
		INNER JOIN (
			SELECT campaignID, MAX(ledgerID) AS maxLedgerID
			FROM `+constants.TableWalletLedger+`
			WHERE createdDate `+op+` ?
			GROUP BY campaignID
		) latest ON latest.campaignID = l.campaignID AND latest.maxLedgerID = l.ledgerID
	`, cutoff).Scan(&row).Error
	return row.Total.Float64, err
}

func (r *RepositoryImpl) BalanceBefore(ctx context.Context, campaignID int64, before time.Time) (float64, error) {
	return r.balanceAt(ctx, campaignID, before, false)
}

func (r *RepositoryImpl) BalanceAsOf(ctx context.Context, campaignID int64, asOf time.Time) (float64, error) {
	return r.balanceAt(ctx, campaignID, asOf, true)
}

func (r *RepositoryImpl) LedgerSumByType(ctx context.Context, campaignID int64, entryType string, from, to time.Time) (float64, error) {
	var row struct {
		Total sql.NullFloat64 `gorm:"column:total"`
	}
	q := r.db.WithContext(ctx).Table(constants.TableWalletLedger).
		Select("SUM(amount) AS total").
		Where("entryType = ? AND createdDate BETWEEN ? AND ?", entryType, from, to)
	if campaignID > 0 {
		q = q.Where("campaignID = ?", campaignID)
	}
	err := q.Find(&row).Error
	return row.Total.Float64, err
}

func (r *RepositoryImpl) WithdrawalSuccessSum(ctx context.Context, campaignID int64, from, to time.Time) (float64, error) {
	var row struct {
		Total sql.NullFloat64 `gorm:"column:total"`
	}
	q := r.db.WithContext(ctx).Table(constants.TableWithdrawal).
		Select("SUM(amount) AS total").
		Where("status = 'SUCCESS' AND completedDate BETWEEN ? AND ?", from, to)
	if campaignID > 0 {
		q = q.Where("campaignID = ?", campaignID)
	}
	err := q.Find(&row).Error
	return row.Total.Float64, err
}

func (r *RepositoryImpl) FeeSum(ctx context.Context, campaignID int64, from, to time.Time) (float64, error) {
	var donationFee struct {
		Total sql.NullFloat64 `gorm:"column:total"`
	}
	dq := r.db.WithContext(ctx).Table(constants.TableDonation).
		Select("SUM(adminFee) AS total").
		Where("paymentStatus = 'PAID' AND updatedDate BETWEEN ? AND ?", from, to)
	if campaignID > 0 {
		dq = dq.Where("campaignID = ?", campaignID)
	}
	if err := dq.Find(&donationFee).Error; err != nil {
		return 0, err
	}

	var withdrawalFee struct {
		Total sql.NullFloat64 `gorm:"column:total"`
	}
	wq := r.db.WithContext(ctx).Table(constants.TableWithdrawal).
		Select("SUM(fee) AS total").
		Where("status = 'SUCCESS' AND completedDate BETWEEN ? AND ?", from, to)
	if campaignID > 0 {
		wq = wq.Where("campaignID = ?", campaignID)
	}
	if err := wq.Find(&withdrawalFee).Error; err != nil {
		return 0, err
	}
	return donationFee.Total.Float64 + withdrawalFee.Total.Float64, nil
}

func (r *RepositoryImpl) CampaignReportRows(ctx context.Context, f report_dto.KantongAmalReportFilter) ([]report_dto.CampaignReportRow, int64, error) {
	base := r.db.WithContext(ctx).Table(constants.TableCampaign + " c").
		Joins(`LEFT JOIN (
			SELECT campaignID, COUNT(DISTINCT donorEmail) AS donorCount, COUNT(*) AS transactionCount
			FROM ` + constants.TableDonation + ` WHERE paymentStatus = 'PAID' GROUP BY campaignID
		) don ON don.campaignID = c.campaignID`)
	if f.CampaignID > 0 {
		base = base.Where("c.campaignID = ?", f.CampaignID)
	}
	if f.Status != "" {
		base = base.Where("c.status = ?", f.Status)
	}
	base = base.Session(&gorm.Session{})

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, limit := f.Page, f.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var out []report_dto.CampaignReportRow
	err := base.Select(`c.campaignID, c.title, c.status, c.targetAmount, c.collectedAmountCache AS collectedAmount,
		COALESCE(don.donorCount, 0) AS donorCount, COALESCE(don.transactionCount, 0) AS transactionCount,
		c.startDate, c.endDate, c.createdDate`).
		Order("c.createdDate DESC").Offset((page - 1) * limit).Limit(limit).Find(&out).Error
	return out, total, err
}

func (r *RepositoryImpl) DonationReportRows(ctx context.Context, f report_dto.KantongAmalReportFilter) ([]report_dto.DonationReportRow, int64, error) {
	base := r.db.WithContext(ctx).Table(constants.TableDonation + " d").
		Joins("JOIN " + constants.TableCampaign + " c ON c.campaignID = d.campaignID")
	if f.CampaignID > 0 {
		base = base.Where("d.campaignID = ?", f.CampaignID)
	}
	if f.Status != "" {
		base = base.Where("d.paymentStatus = ?", f.Status)
	}
	if !f.From.IsZero() && !f.To.IsZero() {
		base = base.Where("d.createdDate BETWEEN ? AND ?", f.From, f.To)
	}
	base = base.Session(&gorm.Session{})

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, limit := f.Page, f.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var out []report_dto.DonationReportRow
	err := base.Select(`d.donationID, c.title AS campaignTitle, d.donorName, d.isAnonymous, d.amount, d.adminFee,
		d.totalAmount, d.paymentStatus, d.gateway, d.createdDate`).
		Order("d.createdDate DESC").Offset((page - 1) * limit).Limit(limit).Find(&out).Error
	return out, total, err
}

func (r *RepositoryImpl) WithdrawalReportRows(ctx context.Context, f report_dto.KantongAmalReportFilter) ([]report_dto.WithdrawalReportRow, int64, error) {
	base := r.db.WithContext(ctx).Table(constants.TableWithdrawal + " w").
		Joins("JOIN " + constants.TableCampaign + " c ON c.campaignID = w.campaignID")
	if f.CampaignID > 0 {
		base = base.Where("w.campaignID = ?", f.CampaignID)
	}
	if f.Status != "" {
		base = base.Where("w.status = ?", f.Status)
	}
	if !f.From.IsZero() && !f.To.IsZero() {
		base = base.Where("w.createdDate BETWEEN ? AND ?", f.From, f.To)
	}
	base = base.Session(&gorm.Session{})

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, limit := f.Page, f.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var out []report_dto.WithdrawalReportRow
	err := base.Select(`w.withdrawalID, w.withdrawalRef, c.title AS campaignTitle, w.amount, w.fee, w.netAmount, w.status,
		w.beneficiaryBankCode, w.beneficiaryAccountNumber, w.createdDate AS requestedDate, w.approvedDate,
		w.executedDate AS processingDate, w.completedDate`).
		Order("w.createdDate DESC").Offset((page - 1) * limit).Limit(limit).Find(&out).Error
	return out, total, err
}

func (r *RepositoryImpl) WithdrawalStatusFunnel(ctx context.Context, campaignID int64) ([]report_dto.WithdrawalStatusFunnel, error) {
	q := r.db.WithContext(ctx).Table(constants.TableWithdrawal).Select("status, COUNT(*) AS cnt")
	if campaignID > 0 {
		q = q.Where("campaignID = ?", campaignID)
	}
	var out []report_dto.WithdrawalStatusFunnel
	err := q.Group("status").Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) DonationPaidTotals(ctx context.Context) (int64, float64, error) {
	var row struct {
		Cnt   int64
		Total sql.NullFloat64 `gorm:"column:total"`
	}
	err := r.db.WithContext(ctx).Table(constants.TableDonation).
		Select("COUNT(*) AS cnt, SUM(amount) AS total").
		Where("paymentStatus = 'PAID'").Find(&row).Error
	return row.Cnt, row.Total.Float64, err
}

func (r *RepositoryImpl) WithdrawalSuccessTotals(ctx context.Context) (int64, float64, error) {
	var row struct {
		Cnt   int64
		Total sql.NullFloat64 `gorm:"column:total"`
	}
	err := r.db.WithContext(ctx).Table(constants.TableWithdrawal).
		Select("COUNT(*) AS cnt, SUM(amount) AS total").
		Where("status = 'SUCCESS'").Find(&row).Error
	return row.Cnt, row.Total.Float64, err
}

func (r *RepositoryImpl) LedgerTotalByType(ctx context.Context, entryType string) (float64, error) {
	var row struct {
		Total sql.NullFloat64 `gorm:"column:total"`
	}
	err := r.db.WithContext(ctx).Table(constants.TableWalletLedger).
		Select("SUM(amount) AS total").
		Where("entryType = ?", entryType).Find(&row).Error
	return row.Total.Float64, err
}

func (r *RepositoryImpl) CreateReconciliationSnapshot(ctx context.Context, p report_dto.ReconciliationSnapshotParams) (int64, error) {
	values := map[string]interface{}{
		"snapshotDate":               p.SnapshotDate,
		"donationPaidCount":          p.DonationPaidCount,
		"donationPaidAmount":         p.DonationPaidAmount,
		"ledgerDonationCreditAmount": p.LedgerDonationCreditAmount,
		"withdrawalSuccessCount":     p.WithdrawalSuccessCount,
		"withdrawalSuccessAmount":    p.WithdrawalSuccessAmount,
		"expectedBalance":            p.ExpectedBalance,
		"gatewayWalletBalance":       p.GatewayWalletBalance,
		"discrepancyAmount":          p.DiscrepancyAmount,
		"hasAnomaly":                 p.HasAnomaly,
		"createdDate":                time.Now(),
	}
	if p.GatewayError != "" {
		values["gatewayError"] = p.GatewayError
	}
	if err := r.db.WithContext(ctx).Table(constants.TableFinanceReconciliationSnapshot).Create(values).Error; err != nil {
		return 0, err
	}
	var newID int64
	err := r.db.WithContext(ctx).Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	return newID, err
}

func (r *RepositoryImpl) ListReconciliationSnapshots(ctx context.Context, q dto.ListQuery) ([]report_dto.ReconciliationSnapshotResponse, int64, error) {
	base := r.db.WithContext(ctx).Table(constants.TableFinanceReconciliationSnapshot).Session(&gorm.Session{})

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []report_dto.ReconciliationSnapshotResponse
	err := base.Order("snapshotID DESC").Offset(q.Offset()).Limit(q.Limit).Find(&out).Error
	return out, total, err
}

func (r *RepositoryImpl) ListFinanceAuditLog(ctx context.Context, f report_dto.FinanceAuditLogFilter) ([]report_dto.FinanceAuditLogItem, int64, error) {
	base := r.db.WithContext(ctx).Table(constants.TableFinanceAuditLog + " l").
		Joins("LEFT JOIN " + constants.TableUser + " u ON u.userID = l.actorUserID")
	if f.Entity != "" {
		base = base.Where("l.entity = ?", f.Entity)
	}
	if f.Action != "" {
		base = base.Where("l.action = ?", f.Action)
	}
	base = base.Session(&gorm.Session{})

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, limit := f.Page, f.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var out []report_dto.FinanceAuditLogItem
	err := base.Select("l.logID, l.actorUserID, u.fullName AS actorName, l.action, l.entity, l.entityID, l.beforeJSON, l.afterJSON, l.metadata, l.createdDate").
		Order("l.logID DESC").Offset((page - 1) * limit).Limit(limit).Find(&out).Error
	return out, total, err
}
