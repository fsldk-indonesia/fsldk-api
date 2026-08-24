package campaign_repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"gorm.io/gorm"

	"fsldk-api/constants"
	"fsldk-api/modules/campaign/campaign_dto"
	"fsldk-api/modules/campaign/campaign_model"
)

const campaignSelectCols = "c.campaignID, c.publicRef, c.slug, c.title, c.categoryID, cat.categoryName, " +
	"c.ownerUserID, u.fullName AS ownerName, c.organizationID, o.organizationName, " +
	"c.story, c.latestUpdate, c.coverImageUrl, c.targetAmount, c.collectedAmountCache, " +
	"c.beneficiaryName, c.beneficiaryBankCode, c.beneficiaryAccountNumber, c.beneficiaryAccountHolder, c.beneficiaryLockedUntil, " +
	"c.startDate, c.endDate, c.status, c.moderationNote, c.isFeatured, c.isAnonymousAllowed, " +
	"c.createdDate, c.createdBy, c.updatedDate, c.updatedBy"

// RepositoryImpl adalah implementasi Repository berbasis GORM.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) baseQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Table(constants.TableCampaign + " c").
		Joins("JOIN " + constants.TableCampaignCategory + " cat ON cat.campaignCategoryID = c.categoryID").
		Joins("JOIN ms_user u ON u.userID = c.ownerUserID").
		Joins("LEFT JOIN ms_organization o ON o.organizationID = c.organizationID")
}

func (r *RepositoryImpl) List(ctx context.Context, f campaign_dto.ListFilter) ([]campaign_model.Campaign, int64, error) {
	q := r.baseQuery(ctx)
	if f.Status != "" {
		q = q.Where("c.status = ?", f.Status)
	}
	if f.CategoryID > 0 {
		q = q.Where("c.categoryID = ?", f.CategoryID)
	}
	if f.OwnerUserID != nil {
		q = q.Where("c.ownerUserID = ?", *f.OwnerUserID)
	}
	if f.Search != "" {
		q = q.Where("c.title LIKE ?", "%"+f.Search+"%")
	}

	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []campaign_model.Campaign
	err := q.Select(campaignSelectCols).Order(f.OrderBy).Limit(f.Limit).Offset(f.Offset).Find(&out).Error
	return out, total, err
}

func (r *RepositoryImpl) findOne(ctx context.Context, where string, arg interface{}) (campaign_model.Campaign, error) {
	var c campaign_model.Campaign
	err := r.baseQuery(ctx).Select(campaignSelectCols).Where(where, arg).Take(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return campaign_model.Campaign{}, ErrNotFound
	}
	return c, err
}

func (r *RepositoryImpl) FindByID(ctx context.Context, id int64) (campaign_model.Campaign, error) {
	return r.findOne(ctx, "c.campaignID = ?", id)
}

func (r *RepositoryImpl) FindBySlug(ctx context.Context, slug string) (campaign_model.Campaign, error) {
	return r.findOne(ctx, "c.slug = ?", slug)
}

func (r *RepositoryImpl) SlugExists(ctx context.Context, slug string, exceptID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table(constants.TableCampaign).
		Where("slug = ? AND campaignID <> ?", slug, exceptID).Count(&count).Error
	return count > 0, err
}

func (r *RepositoryImpl) CategoryExists(ctx context.Context, categoryID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table(constants.TableCampaignCategory).
		Where("campaignCategoryID = ?", categoryID).Count(&count).Error
	return count > 0, err
}

func (r *RepositoryImpl) Categories(ctx context.Context) ([]campaign_model.Category, error) {
	var out []campaign_model.Category
	err := r.db.WithContext(ctx).Table(constants.TableCampaignCategory).Order("sortOrder").Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) Create(ctx context.Context, p campaign_model.CreateParams) (int64, error) {
	values := map[string]interface{}{
		"publicRef":                p.PublicRef,
		"slug":                     p.Slug,
		"title":                    p.Title,
		"categoryID":               p.CategoryID,
		"ownerUserID":              p.OwnerUserID,
		"organizationID":           p.OrganizationID,
		"story":                    p.Story,
		"coverImageUrl":            p.CoverImageUrl,
		"targetAmount":             p.TargetAmount,
		"collectedAmountCache":     0,
		"beneficiaryName":          p.BeneficiaryName,
		"beneficiaryBankCode":      p.BeneficiaryBankCode,
		"beneficiaryAccountNumber": p.BeneficiaryAccountNumber,
		"beneficiaryAccountHolder": p.BeneficiaryAccountHolder,
		"startDate":                p.StartDate,
		"endDate":                  p.EndDate,
		"status":                   constants.CampaignStatusDraft,
		"isAnonymousAllowed":       p.IsAnonymousAllowed,
		"createdDate":              time.Now(),
		"createdBy":                p.CreatedBy,
	}
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table(constants.TableCampaign).Create(values).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) Update(ctx context.Context, id int64, p campaign_model.UpdateParams) error {
	return r.db.WithContext(ctx).Table(constants.TableCampaign).Where("campaignID = ?", id).Updates(map[string]interface{}{
		"slug":                     p.Slug,
		"title":                    p.Title,
		"categoryID":               p.CategoryID,
		"organizationID":           p.OrganizationID,
		"story":                    p.Story,
		"latestUpdate":             p.LatestUpdate,
		"coverImageUrl":            p.CoverImageUrl,
		"targetAmount":             p.TargetAmount,
		"beneficiaryName":          p.BeneficiaryName,
		"beneficiaryBankCode":      p.BeneficiaryBankCode,
		"beneficiaryAccountNumber": p.BeneficiaryAccountNumber,
		"beneficiaryAccountHolder": p.BeneficiaryAccountHolder,
		"startDate":                p.StartDate,
		"endDate":                  p.EndDate,
		"isAnonymousAllowed":       p.IsAnonymousAllowed,
		"updatedDate":              time.Now(),
		"updatedBy":                p.UpdatedBy,
	}).Error
}

func (r *RepositoryImpl) UpdateBeneficiary(ctx context.Context, id int64, p campaign_model.UpdateBeneficiaryParams) error {
	return r.db.WithContext(ctx).Table(constants.TableCampaign).Where("campaignID = ?", id).Updates(map[string]interface{}{
		"beneficiaryName":          p.BeneficiaryName,
		"beneficiaryBankCode":      p.BeneficiaryBankCode,
		"beneficiaryAccountNumber": p.BeneficiaryAccountNumber,
		"beneficiaryAccountHolder": p.BeneficiaryAccountHolder,
		"beneficiaryLockedUntil":   p.LockedUntil,
		"updatedDate":              time.Now(),
	}).Error
}

func (r *RepositoryImpl) UpdateStatus(ctx context.Context, id int64, status string, note sql.NullString, updatedBy int64) error {
	values := map[string]interface{}{
		"status":      status,
		"updatedDate": time.Now(),
		"updatedBy":   updatedBy,
	}
	if note.Valid {
		values["moderationNote"] = note
	}
	return r.db.WithContext(ctx).Table(constants.TableCampaign).Where("campaignID = ?", id).Updates(values).Error
}

func (r *RepositoryImpl) ReplaceImages(ctx context.Context, campaignID int64, urls []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM "+constants.TableCampaignImage+" WHERE campaignID = ?", campaignID).Error; err != nil {
			return err
		}
		for i, u := range urls {
			if err := tx.Table(constants.TableCampaignImage).Create(map[string]interface{}{
				"campaignID": campaignID,
				"imageUrl":   u,
				"sortOrder":  i,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *RepositoryImpl) ListImages(ctx context.Context, campaignID int64) ([]campaign_model.Image, error) {
	var out []campaign_model.Image
	err := r.db.WithContext(ctx).Table(constants.TableCampaignImage).
		Where("campaignID = ?", campaignID).Order("sortOrder").Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) CreateReview(ctx context.Context, p campaign_model.ReviewParams) (int64, error) {
	values := map[string]interface{}{
		"campaignID":     p.CampaignID,
		"reviewerUserID": p.ReviewerUserID,
		"decision":       p.Decision,
		"note":           p.Note,
		"reviewedDate":   time.Now(),
	}
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table(constants.TableCampaignReview).Create(values).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) ListReviews(ctx context.Context, campaignID int64) ([]campaign_model.Review, error) {
	var out []campaign_model.Review
	err := r.db.WithContext(ctx).Table(constants.TableCampaignReview+" r").
		Select("r.reviewID, r.campaignID, r.reviewerUserID, u.fullName AS reviewerName, r.decision, r.note, r.reviewedDate").
		Joins("JOIN ms_user u ON u.userID = r.reviewerUserID").
		Where("r.campaignID = ?", campaignID).
		Order("r.reviewedDate DESC").
		Find(&out).Error
	return out, err
}
