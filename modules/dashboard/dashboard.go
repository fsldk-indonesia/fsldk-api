// Package dashboard menyediakan ringkasan statistik untuk CMS.
package dashboard

import (
	"context"
	"strconv"

	"fsldk-api/base/apperror"
	"fsldk-api/base/httphelper"
	"fsldk-api/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// Summary adalah ringkasan angka pada dashboard.
type Summary struct {
	TotalNews     int `json:"totalNews"`
	PublishedNews int `json:"publishedNews"`
	DraftNews     int `json:"draftNews"`
	TotalUsers    int `json:"totalUsers"`
}

// RecentNews adalah ringkasan berita terbaru.
type RecentNews struct {
	NewsID      int64  `db:"newsID" json:"newsID"`
	NewsTitle   string `db:"newsTitle" json:"newsTitle"`
	IsPublished bool   `db:"isPublished" json:"isPublished"`
}

// Service memuat logika dashboard.
type Service struct{ db *sqlx.DB }

// NewService membuat Service dashboard.
func NewService(db *sqlx.DB) *Service { return &Service{db: db} }

// Summary menghitung ringkasan statistik.
func (s *Service) Summary(ctx context.Context) (Summary, error) {
	var out Summary
	if err := s.db.GetContext(ctx, &out.TotalNews, "SELECT COUNT(*) FROM ms_news"); err != nil {
		return out, err
	}
	if err := s.db.GetContext(ctx, &out.PublishedNews, "SELECT COUNT(*) FROM ms_news WHERE isPublished = 1"); err != nil {
		return out, err
	}
	out.DraftNews = out.TotalNews - out.PublishedNews
	if err := s.db.GetContext(ctx, &out.TotalUsers, "SELECT COUNT(*) FROM ms_user WHERE isActive = 1"); err != nil {
		return out, err
	}
	return out, nil
}

// Recent mengambil berita terbaru.
func (s *Service) Recent(ctx context.Context, limit int) ([]RecentNews, error) {
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	var out []RecentNews
	err := s.db.SelectContext(ctx, &out,
		"SELECT newsID, newsTitle, isPublished FROM ms_news ORDER BY createdDate DESC LIMIT ?", limit)
	if out == nil {
		out = []RecentNews{}
	}
	return out, err
}

// Handler menangani request HTTP dashboard.
type Handler struct{ svc *Service }

// NewHandler membuat Handler dashboard.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// SummaryHandler menangani GET /dashboard/summary.
func (h *Handler) SummaryHandler(c *gin.Context) {
	data, err := h.svc.Summary(c.Request.Context())
	if err != nil {
		httphelper.Error(c, apperror.Internal(""))
		return
	}
	httphelper.Success(c, "", data)
}

// RecentHandler menangani GET /dashboard/recent-news.
func (h *Handler) RecentHandler(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "5"))
	data, err := h.svc.Recent(c.Request.Context(), limit)
	if err != nil {
		httphelper.Error(c, apperror.Internal(""))
		return
	}
	httphelper.Success(c, "", data)
}

// RegisterRoutes mendaftarkan endpoint dashboard.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, mw *middlewares.Middleware) {
	g := rg.Group("/dashboard")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("/summary", h.SummaryHandler)
		g.GET("/recent-news", h.RecentHandler)
	}
}
