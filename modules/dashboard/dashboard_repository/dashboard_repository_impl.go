package dashboard_repository

import (
	"context"

	"fsldk-api/modules/dashboard/dashboard_dto"

	"gorm.io/gorm"
)

// RepositoryImpl adalah implementasi Repository berbasis GORM.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) Summary(ctx context.Context) (dashboard_dto.Summary, error) {
	var out dashboard_dto.Summary
	var totalNews, totalArticles, totalUsers int64

	if err := r.db.WithContext(ctx).Table("ms_news").Count(&totalNews).Error; err != nil {
		return out, err
	}
	if err := r.db.WithContext(ctx).Table("ms_article").Count(&totalArticles).Error; err != nil {
		return out, err
	}
	if err := r.db.WithContext(ctx).Table("ms_user").Where("isActive = 1").Count(&totalUsers).Error; err != nil {
		return out, err
	}

	out.TotalNews = int(totalNews)
	out.TotalArticles = int(totalArticles)
	out.TotalUsers = int(totalUsers)
	return out, nil
}
