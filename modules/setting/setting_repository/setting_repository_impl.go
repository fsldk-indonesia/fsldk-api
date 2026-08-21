package setting_repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"fsldk-api/modules/setting/setting_model"

	"gorm.io/gorm"
)

// RepositoryImpl adalah implementasi Repository berbasis GORM.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) List(ctx context.Context) ([]setting_model.Setting, error) {
	var out []setting_model.Setting
	err := r.db.WithContext(ctx).Table("ms_setting").Order("settingGroup, settingID").Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) FindByID(ctx context.Context, id int64) (setting_model.Setting, error) {
	var s setting_model.Setting
	err := r.db.WithContext(ctx).Table("ms_setting").Where("settingID = ?", id).Take(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return setting_model.Setting{}, ErrNotFound
	}
	return s, err
}

func (r *RepositoryImpl) FindByGroupKey(ctx context.Context, group, key string) (setting_model.Setting, error) {
	var s setting_model.Setting
	err := r.db.WithContext(ctx).Table("ms_setting").
		Where("settingGroup = ? AND settingKey = ?", group, key).Take(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return setting_model.Setting{}, ErrNotFound
	}
	return s, err
}

func (r *RepositoryImpl) Update(ctx context.Context, id int64, value string, updatedBy int64) error {
	return r.db.WithContext(ctx).Table("ms_setting").Where("settingID = ?", id).Updates(map[string]interface{}{
		"settingValue": value,
		"updatedDate":  time.Now(),
		"updatedBy":    sql.NullInt64{Int64: updatedBy, Valid: updatedBy > 0},
	}).Error
}
