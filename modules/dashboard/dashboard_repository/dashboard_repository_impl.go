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
	var totalNews, publishedNews, totalUsers int64

	if err := r.db.WithContext(ctx).Table("ms_news").Count(&totalNews).Error; err != nil {
		return out, err
	}
	if err := r.db.WithContext(ctx).Table("ms_news").Where("isPublished = 1").Count(&publishedNews).Error; err != nil {
		return out, err
	}
	if err := r.db.WithContext(ctx).Table("ms_user").Where("isActive = 1").Count(&totalUsers).Error; err != nil {
		return out, err
	}

	out.TotalNews = int(totalNews)
	out.PublishedNews = int(publishedNews)
	out.DraftNews = out.TotalNews - out.PublishedNews
	out.TotalUsers = int(totalUsers)
	return out, nil
}

func (r *RepositoryImpl) RecentNews(ctx context.Context, limit int) ([]dashboard_dto.RecentNews, error) {
	var out []dashboard_dto.RecentNews
	err := r.db.WithContext(ctx).Table("ms_news").
		Select("newsID, newsTitle, isPublished").
		Order("createdDate DESC").Limit(limit).Find(&out).Error
	if out == nil {
		out = []dashboard_dto.RecentNews{}
	}
	return out, err
}
