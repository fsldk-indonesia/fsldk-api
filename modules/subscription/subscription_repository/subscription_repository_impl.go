package subscription_repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"fsldk-api/base/dto"
	"fsldk-api/modules/subscription/subscription_model"

	"gorm.io/gorm"
)

// subscriberSortColumns whitelists the fields sortable via dto.ListQuery.Sort
// (mis. "-subscribedDate", "email") — lihat dto.ListQuery.OrderBy.
var subscriberSortColumns = map[string]string{
	"email":          "email",
	"isactive":       "isActive",
	"subscribeddate": "subscribedDate",
	"createddate":    "createdDate",
}

type repositoryImpl struct {
	db *gorm.DB
}

// NewRepository creates a new instance of subscription Repository.
func NewRepository(db *gorm.DB) Repository {
	return &repositoryImpl{db: db}
}

func (r *repositoryImpl) FindByEmail(ctx context.Context, email string) (*subscription_model.Subscriber, error) {
	var sub subscription_model.Subscriber
	err := r.db.WithContext(ctx).First(&sub, "email = ?", email).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *repositoryImpl) FindByID(ctx context.Context, id int64) (*subscription_model.Subscriber, error) {
	var sub subscription_model.Subscriber
	err := r.db.WithContext(ctx).First(&sub, "subscriberID = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *repositoryImpl) FindAll(ctx context.Context, q dto.ListQuery, isActive *bool, from, to string) ([]subscription_model.Subscriber, int, error) {
	var subs []subscription_model.Subscriber
	var total int64

	db := r.db.WithContext(ctx).Model(&subscription_model.Subscriber{})

	if q.Search != "" {
		db = db.Where("email LIKE ?", "%"+strings.TrimSpace(q.Search)+"%")
	}
	if isActive != nil {
		db = db.Where("isActive = ?", *isActive)
	}
	if parsed, err := time.Parse("2006-01-02", from); err == nil {
		db = db.Where("subscribedDate >= ?", parsed)
	}
	if parsed, err := time.Parse("2006-01-02", to); err == nil {
		db = db.Where("subscribedDate < ?", parsed.AddDate(0, 0, 1))
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	db = db.Order(q.OrderBy(subscriberSortColumns, "subscribedDate DESC"))

	if err := db.Limit(q.Limit).Offset(q.Offset()).Find(&subs).Error; err != nil {
		return nil, 0, err
	}

	return subs, int(total), nil
}

func (r *repositoryImpl) Create(ctx context.Context, sub *subscription_model.Subscriber) error {
	return r.db.WithContext(ctx).Create(sub).Error
}

func (r *repositoryImpl) Update(ctx context.Context, sub *subscription_model.Subscriber) error {
	return r.db.WithContext(ctx).Save(sub).Error
}

func (r *repositoryImpl) EmailExistsExcluding(ctx context.Context, email string, excludeID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&subscription_model.Subscriber{}).
		Where("email = ? AND subscriberID != ?", email, excludeID).Count(&count).Error
	return count > 0, err
}

func (r *repositoryImpl) Delete(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Delete(&subscription_model.Subscriber{}, "subscriberID = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *repositoryImpl) DeleteMany(ctx context.Context, ids []int64) error {
	return r.db.WithContext(ctx).Delete(&subscription_model.Subscriber{}, "subscriberID IN ?", ids).Error
}
