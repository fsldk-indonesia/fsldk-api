package news_service

import (
	"context"
	"fmt"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/ptr"
	"fsldk-api/base/slug"
	"fsldk-api/modules/news/news_dto"
	"fsldk-api/modules/news/news_model"
	"fsldk-api/modules/news/news_repository"
)

var sortColumns = map[string]string{
	"newsTitle":     "n.newsTitle",
	"publishedDate": "n.publishedDate",
	"createdDate":   "n.createdDate",
	"viewCount":     "n.viewCount",
}

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct{ repo news_repository.Repository }

// NewService membuat Service berita.
func NewService(repo news_repository.Repository) Service { return &ServiceImpl{repo: repo} }

func (s *ServiceImpl) PublicList(ctx context.Context, q dto.ListQuery, categorySlug string) ([]news_model.News, int, error) {
	return s.list(ctx, news_dto.Filter{
		Search: q.Search, CategorySlug: categorySlug, PublishedOnly: true,
		Limit: q.Limit, Offset: q.Offset(), OrderBy: q.OrderBy(sortColumns, "n.publishedDate DESC"),
	})
}

func (s *ServiceImpl) CMSList(ctx context.Context, q dto.ListQuery, status string, categoryID int64) ([]news_model.News, int, error) {
	return s.list(ctx, news_dto.Filter{
		Search: q.Search, Status: status, CategoryID: categoryID,
		Limit: q.Limit, Offset: q.Offset(), OrderBy: q.OrderBy(sortColumns, "n.createdDate DESC"),
	})
}

func (s *ServiceImpl) list(ctx context.Context, f news_dto.Filter) ([]news_model.News, int, error) {
	data, total, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	if data == nil {
		data = []news_model.News{}
	}
	return data, int(total), nil
}

func (s *ServiceImpl) PublicDetail(ctx context.Context, slugStr string) (news_model.News, error) {
	n, err := s.repo.FindBySlug(ctx, slugStr)
	if err != nil || !n.IsPublished {
		return news_model.News{}, apperror.NotFound("Berita tidak ditemukan")
	}
	_ = s.repo.IncrementView(ctx, n.NewsID)
	n.ViewCount++
	return n, nil
}

func (s *ServiceImpl) Get(ctx context.Context, id int64) (news_model.News, error) {
	n, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return news_model.News{}, apperror.NotFound("Berita tidak ditemukan")
	}
	return n, nil
}

func (s *ServiceImpl) Featured(ctx context.Context, limit int) ([]news_model.News, error) {
	if limit <= 0 {
		limit = 5
	}
	data, err := s.repo.Featured(ctx, limit)
	if err != nil {
		return nil, apperror.Internal("")
	}
	if data == nil {
		data = []news_model.News{}
	}
	return data, nil
}

func (s *ServiceImpl) Categories(ctx context.Context) ([]news_model.Category, error) {
	data, err := s.repo.Categories(ctx)
	if err != nil {
		return nil, apperror.Internal("")
	}
	if data == nil {
		data = []news_model.Category{}
	}
	return data, nil
}

func (s *ServiceImpl) Create(ctx context.Context, req news_dto.Request, authorID int64, canPublish bool) (news_model.News, error) {
	entity := s.fromRequest(req)
	published := req.Status == "published"
	if published && !canPublish {
		return news_model.News{}, apperror.Forbidden("Anda tidak memiliki hak untuk mempublikasikan berita")
	}
	entity.IsPublished = published

	slugStr, err := s.uniqueSlug(ctx, req.NewsTitle, 0)
	if err != nil {
		return news_model.News{}, err
	}
	entity.NewsSlug = slugStr

	id, err := s.repo.Create(ctx, entity, authorID)
	if err != nil {
		return news_model.News{}, apperror.Internal("Gagal menyimpan berita")
	}
	return s.Get(ctx, id)
}

func (s *ServiceImpl) Update(ctx context.Context, id int64, req news_dto.Request, updatedBy int64) (news_model.News, error) {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return news_model.News{}, apperror.NotFound("Berita tidak ditemukan")
	}
	entity := s.fromRequest(req)
	slugStr := existing.NewsSlug
	if existing.NewsTitle != req.NewsTitle {
		slugStr, err = s.uniqueSlug(ctx, req.NewsTitle, id)
		if err != nil {
			return news_model.News{}, err
		}
	}
	entity.NewsSlug = slugStr
	if err := s.repo.Update(ctx, id, entity, updatedBy); err != nil {
		return news_model.News{}, apperror.Internal("")
	}
	return s.Get(ctx, id)
}

func (s *ServiceImpl) SetPublished(ctx context.Context, id int64, published bool, updatedBy int64) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return apperror.NotFound("Berita tidak ditemukan")
	}
	if err := s.repo.SetPublished(ctx, id, published, updatedBy); err != nil {
		return apperror.Internal("")
	}
	return nil
}

func (s *ServiceImpl) SetFeatured(ctx context.Context, id int64, featured bool, updatedBy int64) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return apperror.NotFound("Berita tidak ditemukan")
	}
	if err := s.repo.SetFeatured(ctx, id, featured, updatedBy); err != nil {
		return apperror.Internal("")
	}
	return nil
}

func (s *ServiceImpl) Delete(ctx context.Context, id int64) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return apperror.NotFound("Berita tidak ditemukan")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return apperror.Internal("")
	}
	return nil
}

func (s *ServiceImpl) fromRequest(req news_dto.Request) news_model.News {
	return news_model.News{
		NewsTitle:     req.NewsTitle,
		NewsExcerpt:   ptr.Str(req.NewsExcerpt),
		NewsContent:   req.NewsContent,
		NewsImage:     ptr.Str(req.NewsImage),
		NewsPublisher: ptr.Str(req.NewsPublisher),
		NewsReporter:  ptr.Str(req.NewsReporter),
		NewsEditor:    ptr.Str(req.NewsEditor),
		CategoryID:    req.CategoryID,
		IsFeatured:    req.IsFeatured,
	}
}

func (s *ServiceImpl) uniqueSlug(ctx context.Context, title string, exceptID int64) (string, error) {
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
