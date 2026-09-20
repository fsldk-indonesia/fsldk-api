package welcomepopup_repository

import (
	"context"
	"time"

	"fsldk-api/modules/welcomepopup/welcomepopup_model"

	"gorm.io/gorm"
)

// RepositoryImpl adalah implementasi Repository berbasis GORM.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) Get(ctx context.Context) (welcomepopup_model.WelcomePopup, error) {
	var w welcomepopup_model.WelcomePopup
	err := r.db.WithContext(ctx).Table("ms_welcome_popup").
		Where("id = ?", welcomepopup_model.SingletonID).Take(&w).Error
	return w, err
}

func (r *RepositoryImpl) Update(ctx context.Context, popup welcomepopup_model.WelcomePopup) error {
	return r.db.WithContext(ctx).Table("ms_welcome_popup").Where("id = ?", welcomepopup_model.SingletonID).
		Updates(map[string]interface{}{
			"isEnabled":   popup.IsEnabled,
			"htmlContent": popup.HTMLContent,
			"jsContent":   popup.JSContent,
			"cssContent":  popup.CSSContent,
			"updatedDate": time.Now(),
			"updatedBy":   popup.UpdatedBy,
		}).Error
}
