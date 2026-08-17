package article_service

import (
	"context"
	"fmt"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/ptr"
	"fsldk-api/base/slug"
	"fsldk-api/modules/article/article_dto"
	"fsldk-api/modules/article/article_model"
	"fsldk-api/modules/article/article_repository"
)

var sortColumns = map[string]string{
	"articleTitle":  "a.articleTitle",
	"publishedDate": "a.publishedDate",
	"createdDate":   "a.createdDate",
}

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct {
	repo    article_repository.Repository
	comment CommentCleaner
}

// NewService membuat Service artikel.
func NewService(repo article_repository.Repository, comment CommentCleaner) Service {
	return &ServiceImpl{repo: repo, comment: comment}
}

func (s *ServiceImpl) PublicList(ctx context.Context, q dto.ListQuery, categorySlug string) ([]article_model.Article, int, error) {
	return s.list(ctx, article_dto.Filter{
		Search: q.Search, CategorySlug: categorySlug, PublishedOnly: true,
		Limit: q.Limit, Offset: q.Offset(), OrderBy: q.OrderBy(sortColumns, "a.publishedDate DESC"),
	})
}

func (s *ServiceImpl) CMSList(ctx context.Context, q dto.ListQuery, status string, categoryID int64) ([]article_model.Article, int, error) {
	return s.list(ctx, article_dto.Filter{
		Search: q.Search, Status: status, CategoryID: categoryID,
		Limit: q.Limit, Offset: q.Offset(), OrderBy: q.OrderBy(sortColumns, "a.createdDate DESC"),
	})
}

func (s *ServiceImpl) list(ctx context.Context, f article_dto.Filter) ([]article_model.Article, int, error) {
	data, total, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	if data == nil {
		data = []article_model.Article{}
	}
	return data, int(total), nil
}

func (s *ServiceImpl) PublicDetail(ctx context.Context, slugStr string) (article_model.Article, error) {
	a, err := s.repo.FindBySlug(ctx, slugStr)
	if err != nil || !a.IsPublished {
		return article_model.Article{}, apperror.NotFound("Artikel tidak ditemukan")
	}
	return a, nil
}

func (s *ServiceImpl) Get(ctx context.Context, id int64) (article_model.Article, error) {
	a, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return article_model.Article{}, apperror.NotFound("Artikel tidak ditemukan")
	}
	return a, nil
}

func (s *ServiceImpl) Categories(ctx context.Context) ([]article_model.Category, error) {
	data, err := s.repo.Categories(ctx)
	if err != nil {
		return nil, apperror.Internal("")
	}
	if data == nil {
		data = []article_model.Category{}
	}
	return data, nil
}

func (s *ServiceImpl) Create(ctx context.Context, req article_dto.Request, authorID int64, canPublish bool) (article_model.Article, error) {
	entity := s.fromRequest(req)
	published := req.Status == "published"
	if published && !canPublish {
		return article_model.Article{}, apperror.Forbidden("Anda tidak memiliki hak untuk mempublikasikan artikel")
	}
	entity.IsPublished = published
	slugStr, err := s.uniqueSlug(ctx, req.ArticleTitle, 0)
	if err != nil {
		return article_model.Article{}, err
	}
	entity.ArticleSlug = slugStr
	id, err := s.repo.Create(ctx, entity, authorID)
	if err != nil {
		return article_model.Article{}, apperror.Internal("Gagal menyimpan artikel")
	}
	return s.Get(ctx, id)
}

func (s *ServiceImpl) Update(ctx context.Context, id int64, req article_dto.Request, updatedBy int64) (article_model.Article, error) {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return article_model.Article{}, apperror.NotFound("Artikel tidak ditemukan")
	}
	entity := s.fromRequest(req)
	slugStr := existing.ArticleSlug
	if existing.ArticleTitle != req.ArticleTitle {
		slugStr, err = s.uniqueSlug(ctx, req.ArticleTitle, id)
		if err != nil {
			return article_model.Article{}, err
		}
	}
	entity.ArticleSlug = slugStr
	if err := s.repo.Update(ctx, id, entity, updatedBy); err != nil {
		return article_model.Article{}, apperror.Internal("")
	}
	return s.Get(ctx, id)
}

func (s *ServiceImpl) SetPublished(ctx context.Context, id int64, published bool, updatedBy int64) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return apperror.NotFound("Artikel tidak ditemukan")
	}
	if err := s.repo.SetPublished(ctx, id, published, updatedBy); err != nil {
		return apperror.Internal("")
	}
	return nil
}

func (s *ServiceImpl) Delete(ctx context.Context, id int64) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return apperror.NotFound("Artikel tidak ditemukan")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return apperror.Internal("")
	}
	// Best-effort: ms_comment has no FK to ms_article (see comment techspec
	// §3.1a), so comments aren't cascaded by the database — clean them up
	// explicitly. A failure here does not roll back the article delete.
	_ = s.comment.DeleteByContent(ctx, "article", id)
	return nil
}

func (s *ServiceImpl) fromRequest(req article_dto.Request) article_model.Article {
	return article_model.Article{
		ArticleTitle:  req.ArticleTitle,
		ArticleIntro:  req.ArticleIntro,
		ArticleImage:  ptr.Str(req.ArticleImage),
		ArticleWriter: ptr.Str(req.ArticleWriter),
		ArticleEditor: ptr.Str(req.ArticleEditor),
		ArticlePdf:    ptr.Str(req.ArticlePdf),
		CategoryID:    req.CategoryID,
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
