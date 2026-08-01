package news

import (
	"context"
	"database/sql"
	"fmt"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/slug"
)

var sortColumns = map[string]string{
	"newsTitle":     "n.newsTitle",
	"publishedDate": "n.publishedDate",
	"createdDate":   "n.createdDate",
	"viewCount":     "n.viewCount",
}

// Service memuat logika bisnis berita.
type Service struct{ repo Repository }

// NewService membuat Service berita.
func NewService(repo Repository) *Service { return &Service{repo: repo} }

// PublicList mengembalikan berita terpublikasi (untuk Landing Page).
func (s *Service) PublicList(ctx context.Context, q dto.ListQuery, categorySlug string) ([]News, int, error) {
	return s.list(ctx, Filter{
		Search:        q.Search,
		CategorySlug:  categorySlug,
		PublishedOnly: true,
		Limit:         q.Limit,
		Offset:        q.Offset(),
		OrderBy:       q.OrderBy(sortColumns, "n.publishedDate DESC"),
	})
}

// CMSList mengembalikan berita untuk pengelolaan (semua status).
func (s *Service) CMSList(ctx context.Context, q dto.ListQuery, status string, categoryID int64) ([]News, int, error) {
	return s.list(ctx, Filter{
		Search:     q.Search,
		Status:     status,
		CategoryID: categoryID,
		Limit:      q.Limit,
		Offset:     q.Offset(),
		OrderBy:    q.OrderBy(sortColumns, "n.createdDate DESC"),
	})
}

func (s *Service) list(ctx context.Context, f Filter) ([]News, int, error) {
	data, total, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	if data == nil {
		data = []News{}
	}
	return data, total, nil
}

// PublicDetail mengambil berita terpublikasi berdasarkan slug & menambah view.
func (s *Service) PublicDetail(ctx context.Context, slugStr string) (News, error) {
	n, err := s.repo.FindBySlug(ctx, slugStr)
	if err != nil || !n.IsPublished {
		return News{}, apperror.NotFound("Berita tidak ditemukan")
	}
	_ = s.repo.IncrementView(ctx, n.NewsID)
	n.ViewCount++
	return n, nil
}

// Get mengambil berita untuk pengelolaan (CMS).
func (s *Service) Get(ctx context.Context, id int64) (News, error) {
	n, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return News{}, apperror.NotFound("Berita tidak ditemukan")
	}
	return n, nil
}

// Featured mengembalikan berita unggulan.
func (s *Service) Featured(ctx context.Context, limit int) ([]News, error) {
	if limit <= 0 {
		limit = 5
	}
	data, err := s.repo.Featured(ctx, limit)
	if err != nil {
		return nil, apperror.Internal("")
	}
	if data == nil {
		data = []News{}
	}
	return data, nil
}

// Categories mengembalikan daftar kategori berita.
func (s *Service) Categories(ctx context.Context) ([]Category, error) {
	data, err := s.repo.Categories(ctx)
	if err != nil {
		return nil, apperror.Internal("")
	}
	if data == nil {
		data = []Category{}
	}
	return data, nil
}

// Create membuat berita baru. canPublish menentukan apakah aktor boleh publish.
func (s *Service) Create(ctx context.Context, req Request, authorID int64, canPublish bool) (News, error) {
	entity := s.fromRequest(req)
	published := req.Status == "published"
	if published && !canPublish {
		return News{}, apperror.Forbidden("Anda tidak memiliki hak untuk mempublikasikan berita")
	}
	entity.IsPublished = published

	slugStr, err := s.uniqueSlug(ctx, req.NewsTitle, 0)
	if err != nil {
		return News{}, err
	}
	entity.NewsSlug = slugStr

	id, err := s.repo.Create(ctx, entity, authorID)
	if err != nil {
		return News{}, apperror.Internal("Gagal menyimpan berita")
	}
	return s.Get(ctx, id)
}

// Update memperbarui berita.
func (s *Service) Update(ctx context.Context, id int64, req Request, updatedBy int64) (News, error) {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return News{}, apperror.NotFound("Berita tidak ditemukan")
	}
	entity := s.fromRequest(req)
	slugStr := existing.NewsSlug
	if existing.NewsTitle != req.NewsTitle {
		slugStr, err = s.uniqueSlug(ctx, req.NewsTitle, id)
		if err != nil {
			return News{}, err
		}
	}
	entity.NewsSlug = slugStr
	if err := s.repo.Update(ctx, id, entity, updatedBy); err != nil {
		return News{}, apperror.Internal("")
	}
	return s.Get(ctx, id)
}

// SetPublished mengubah status publikasi (butuh hak publish).
func (s *Service) SetPublished(ctx context.Context, id int64, published bool, updatedBy int64) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return apperror.NotFound("Berita tidak ditemukan")
	}
	if err := s.repo.SetPublished(ctx, id, published, updatedBy); err != nil {
		return apperror.Internal("")
	}
	return nil
}

// SetFeatured mengubah status unggulan.
func (s *Service) SetFeatured(ctx context.Context, id int64, featured bool, updatedBy int64) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return apperror.NotFound("Berita tidak ditemukan")
	}
	if err := s.repo.SetFeatured(ctx, id, featured, updatedBy); err != nil {
		return apperror.Internal("")
	}
	return nil
}

// Delete menghapus berita.
func (s *Service) Delete(ctx context.Context, id int64) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return apperror.NotFound("Berita tidak ditemukan")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return apperror.Internal("")
	}
	return nil
}

func (s *Service) fromRequest(req Request) News {
	return News{
		NewsTitle:   req.NewsTitle,
		NewsExcerpt: sql.NullString{String: req.NewsExcerpt, Valid: req.NewsExcerpt != ""},
		NewsContent: req.NewsContent,
		NewsImage:   sql.NullString{String: req.NewsImage, Valid: req.NewsImage != ""},
		CategoryID:  req.CategoryID,
		IsFeatured:  req.IsFeatured,
	}
}

func (s *Service) uniqueSlug(ctx context.Context, title string, exceptID int64) (string, error) {
	base := slug.Make(title)
	candidate := base
	for i := 2; i < 100; i++ {
		exists, err := s.repo.SlugExists(ctx, candidate, exceptID)
		if err != nil {
			return "", apperror.Internal("")
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
	return fmt.Sprintf("%s-%d", base, exceptID), nil
}
