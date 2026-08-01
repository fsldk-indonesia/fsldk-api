package article

import (
	"context"
	"database/sql"
	"fmt"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/slug"
)

var sortColumns = map[string]string{
	"articleTitle":  "a.articleTitle",
	"publishedDate": "a.publishedDate",
	"createdDate":   "a.createdDate",
}

// Service memuat logika bisnis artikel.
type Service struct{ repo Repository }

// NewService membuat Service artikel.
func NewService(repo Repository) *Service { return &Service{repo: repo} }

// PublicList mengembalikan artikel terpublikasi.
func (s *Service) PublicList(ctx context.Context, q dto.ListQuery, categorySlug string) ([]Article, int, error) {
	return s.list(ctx, Filter{
		Search: q.Search, CategorySlug: categorySlug, PublishedOnly: true,
		Limit: q.Limit, Offset: q.Offset(), OrderBy: q.OrderBy(sortColumns, "a.publishedDate DESC"),
	})
}

// CMSList mengembalikan artikel untuk pengelolaan (semua status).
func (s *Service) CMSList(ctx context.Context, q dto.ListQuery, status string, categoryID int64) ([]Article, int, error) {
	return s.list(ctx, Filter{
		Search: q.Search, Status: status, CategoryID: categoryID,
		Limit: q.Limit, Offset: q.Offset(), OrderBy: q.OrderBy(sortColumns, "a.createdDate DESC"),
	})
}

func (s *Service) list(ctx context.Context, f Filter) ([]Article, int, error) {
	data, total, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	if data == nil {
		data = []Article{}
	}
	return data, total, nil
}

// PublicDetail mengambil artikel terpublikasi berdasarkan slug.
func (s *Service) PublicDetail(ctx context.Context, slugStr string) (Article, error) {
	a, err := s.repo.FindBySlug(ctx, slugStr)
	if err != nil || !a.IsPublished {
		return Article{}, apperror.NotFound("Artikel tidak ditemukan")
	}
	return a, nil
}

// Get mengambil artikel untuk pengelolaan (CMS).
func (s *Service) Get(ctx context.Context, id int64) (Article, error) {
	a, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return Article{}, apperror.NotFound("Artikel tidak ditemukan")
	}
	return a, nil
}

// Categories mengembalikan daftar kategori artikel.
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

// Create membuat artikel baru.
func (s *Service) Create(ctx context.Context, req Request, authorID int64, canPublish bool) (Article, error) {
	entity := s.fromRequest(req)
	published := req.Status == "published"
	if published && !canPublish {
		return Article{}, apperror.Forbidden("Anda tidak memiliki hak untuk mempublikasikan artikel")
	}
	entity.IsPublished = published
	slugStr, err := s.uniqueSlug(ctx, req.ArticleTitle, 0)
	if err != nil {
		return Article{}, err
	}
	entity.ArticleSlug = slugStr
	id, err := s.repo.Create(ctx, entity, authorID)
	if err != nil {
		return Article{}, apperror.Internal("Gagal menyimpan artikel")
	}
	return s.Get(ctx, id)
}

// Update memperbarui artikel.
func (s *Service) Update(ctx context.Context, id int64, req Request, updatedBy int64) (Article, error) {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return Article{}, apperror.NotFound("Artikel tidak ditemukan")
	}
	entity := s.fromRequest(req)
	slugStr := existing.ArticleSlug
	if existing.ArticleTitle != req.ArticleTitle {
		slugStr, err = s.uniqueSlug(ctx, req.ArticleTitle, id)
		if err != nil {
			return Article{}, err
		}
	}
	entity.ArticleSlug = slugStr
	if err := s.repo.Update(ctx, id, entity, updatedBy); err != nil {
		return Article{}, apperror.Internal("")
	}
	return s.Get(ctx, id)
}

// SetPublished mengubah status publikasi artikel.
func (s *Service) SetPublished(ctx context.Context, id int64, published bool, updatedBy int64) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return apperror.NotFound("Artikel tidak ditemukan")
	}
	if err := s.repo.SetPublished(ctx, id, published, updatedBy); err != nil {
		return apperror.Internal("")
	}
	return nil
}

// Delete menghapus artikel.
func (s *Service) Delete(ctx context.Context, id int64) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return apperror.NotFound("Artikel tidak ditemukan")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return apperror.Internal("")
	}
	return nil
}

func (s *Service) fromRequest(req Request) Article {
	return Article{
		ArticleTitle:   req.ArticleTitle,
		ArticleExcerpt: sql.NullString{String: req.ArticleExcerpt, Valid: req.ArticleExcerpt != ""},
		ArticleContent: req.ArticleContent,
		ArticleImage:   sql.NullString{String: req.ArticleImage, Valid: req.ArticleImage != ""},
		CategoryID:     req.CategoryID,
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
