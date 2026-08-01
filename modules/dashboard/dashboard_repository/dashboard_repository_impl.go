package dashboard_repository

import (
	"context"

	"fsldk-api/modules/dashboard/dashboard_dto"

	"github.com/jmoiron/sqlx"
)

// RepositoryImpl adalah implementasi Repository berbasis sqlx.
type RepositoryImpl struct{ db *sqlx.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *sqlx.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) Summary(ctx context.Context) (dashboard_dto.Summary, error) {
	var out dashboard_dto.Summary
	if err := r.db.GetContext(ctx, &out.TotalNews, "SELECT COUNT(*) FROM ms_news"); err != nil {
		return out, err
	}
	if err := r.db.GetContext(ctx, &out.PublishedNews, "SELECT COUNT(*) FROM ms_news WHERE isPublished = 1"); err != nil {
		return out, err
	}
	out.DraftNews = out.TotalNews - out.PublishedNews
	if err := r.db.GetContext(ctx, &out.TotalUsers, "SELECT COUNT(*) FROM ms_user WHERE isActive = 1"); err != nil {
		return out, err
	}
	return out, nil
}

func (r *RepositoryImpl) RecentNews(ctx context.Context, limit int) ([]dashboard_dto.RecentNews, error) {
	var out []dashboard_dto.RecentNews
	err := r.db.SelectContext(ctx, &out,
		"SELECT newsID, newsTitle, isPublished FROM ms_news ORDER BY createdDate DESC LIMIT ?", limit)
	if out == nil {
		out = []dashboard_dto.RecentNews{}
	}
	return out, err
}
