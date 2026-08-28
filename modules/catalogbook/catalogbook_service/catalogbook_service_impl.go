package catalogbook_service

import (
	"context"
	"errors"
	"fmt"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/ptr"
	"fsldk-api/base/slug"
	"fsldk-api/modules/catalogbook/catalogbook_dto"
	"fsldk-api/modules/catalogbook/catalogbook_model"
	"fsldk-api/modules/catalogbook/catalogbook_repository"
)

var sortColumns = map[string]string{
	"bookTitle":     "b.bookTitle",
	"authorName":    "b.authorName",
	"publisherName": "b.publisherName",
	"favoriteCount": "b.favoriteCount",
	"createdDate":   "b.createdDate",
}

// publicSortColumns are the 3 fixed sort options for the public listing
// (newest/popular/title), distinct from the CMS's free-form whitelist.
var publicSortColumns = map[string]string{
	"popular": "b.favoriteCount DESC",
	"title":   "b.bookTitle ASC",
	"newest":  "b.createdDate DESC",
}

// FileDeleter is the narrow slice of pkg/upload.Uploader this service
// depends on — accepting an interface (not the concrete type) keeps
// catalogbook_service decoupled from the upload package's implementation.
type FileDeleter interface {
	DeleteFile(publicURL string) error
}

// ServiceImpl is the Service implementation.
type ServiceImpl struct {
	repo    catalogbook_repository.Repository
	upload  FileDeleter
	comment CommentCleaner
}

// NewService creates the catalogbook Service.
func NewService(repo catalogbook_repository.Repository, upload FileDeleter, comment CommentCleaner) Service {
	return &ServiceImpl{repo: repo, upload: upload, comment: comment}
}

func (s *ServiceImpl) PublicList(ctx context.Context, q dto.ListQuery, f catalogbook_dto.Filter, sort string) ([]catalogbook_model.CatalogBook, int, error) {
	f.ActiveOnly = true
	f.Limit = q.Limit
	f.Offset = q.Offset()
	orderBy, ok := publicSortColumns[sort]
	if !ok {
		orderBy = publicSortColumns["newest"]
	}
	f.OrderBy = orderBy
	if f.Search == "" {
		f.Search = q.Search
	}
	return s.list(ctx, f)
}

func (s *ServiceImpl) CMSList(ctx context.Context, q dto.ListQuery, f catalogbook_dto.Filter) ([]catalogbook_model.CatalogBook, int, error) {
	f.ActiveOnly = false
	f.Limit = q.Limit
	f.Offset = q.Offset()
	f.OrderBy = q.OrderBy(sortColumns, "b.createdDate DESC")
	if f.Search == "" {
		f.Search = q.Search
	}
	return s.list(ctx, f)
}

func (s *ServiceImpl) list(ctx context.Context, f catalogbook_dto.Filter) ([]catalogbook_model.CatalogBook, int, error) {
	data, total, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	if data == nil {
		data = []catalogbook_model.CatalogBook{}
	}
	return data, int(total), nil
}

func (s *ServiceImpl) PublicDetail(ctx context.Context, slugStr string) (catalogbook_model.CatalogBook, error) {
	b, err := s.repo.FindBySlug(ctx, slugStr)
	if err != nil || !b.IsActive {
		return catalogbook_model.CatalogBook{}, apperror.NotFound("Buku tidak ditemukan")
	}
	return b, nil
}

func (s *ServiceImpl) Get(ctx context.Context, id int64) (catalogbook_model.CatalogBook, error) {
	b, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return catalogbook_model.CatalogBook{}, apperror.NotFound("Buku tidak ditemukan")
	}
	return b, nil
}

func (s *ServiceImpl) Categories(ctx context.Context) ([]catalogbook_model.BookCategory, error) {
	data, err := s.repo.Categories(ctx)
	if err != nil {
		return nil, apperror.Internal("")
	}
	if data == nil {
		data = []catalogbook_model.BookCategory{}
	}
	return data, nil
}

func (s *ServiceImpl) Languages(ctx context.Context) ([]catalogbook_model.Language, error) {
	data, err := s.repo.Languages(ctx)
	if err != nil {
		return nil, apperror.Internal("")
	}
	if data == nil {
		data = []catalogbook_model.Language{}
	}
	return data, nil
}

func (s *ServiceImpl) AuthorTypes(ctx context.Context) ([]catalogbook_model.AuthorType, error) {
	data, err := s.repo.AuthorTypes(ctx)
	if err != nil {
		return nil, apperror.Internal("")
	}
	if data == nil {
		data = []catalogbook_model.AuthorType{}
	}
	return data, nil
}

func (s *ServiceImpl) AvailabilityTypes(ctx context.Context) ([]catalogbook_model.AvailabilityType, error) {
	data, err := s.repo.AvailabilityTypes(ctx)
	if err != nil {
		return nil, apperror.Internal("")
	}
	if data == nil {
		data = []catalogbook_model.AvailabilityType{}
	}
	return data, nil
}

func (s *ServiceImpl) Create(ctx context.Context, req catalogbook_dto.Request, actorID int64) (catalogbook_model.CatalogBook, error) {
	entity := s.fromRequest(req)
	slugStr, err := s.uniqueSlug(ctx, req.BookTitle, 0)
	if err != nil {
		return catalogbook_model.CatalogBook{}, err
	}
	entity.BookSlug = slugStr
	id, err := s.repo.Create(ctx, entity, actorID)
	if err != nil {
		return catalogbook_model.CatalogBook{}, apperror.Internal("Gagal menyimpan buku")
	}
	return s.Get(ctx, id)
}

func (s *ServiceImpl) Update(ctx context.Context, id int64, req catalogbook_dto.Request, actorID int64) (catalogbook_model.CatalogBook, error) {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return catalogbook_model.CatalogBook{}, apperror.NotFound("Buku tidak ditemukan")
	}
	entity := s.fromRequest(req)
	slugStr := existing.BookSlug
	if existing.BookTitle != req.BookTitle {
		slugStr, err = s.uniqueSlug(ctx, req.BookTitle, id)
		if err != nil {
			return catalogbook_model.CatalogBook{}, err
		}
	}
	entity.BookSlug = slugStr
	if err := s.repo.Update(ctx, id, entity, actorID); err != nil {
		return catalogbook_model.CatalogBook{}, apperror.Internal("")
	}
	// Best-effort cleanup of the old cover/PDF ONLY after the DB update
	// succeeds and the field actually changed — doing this before the update
	// could delete the file from disk even if the DB update then fails.
	if existing.CoverImage != nil && entity.CoverImage != nil && *existing.CoverImage != *entity.CoverImage {
		_ = s.upload.DeleteFile(*existing.CoverImage)
	}
	if existing.BookPdf != nil && entity.BookPdf != nil && *existing.BookPdf != *entity.BookPdf {
		_ = s.upload.DeleteFile(*existing.BookPdf)
	}
	return s.Get(ctx, id)
}

func (s *ServiceImpl) SetActive(ctx context.Context, id int64, isActive bool, actorID int64) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return apperror.NotFound("Buku tidak ditemukan")
	}
	if err := s.repo.SetActive(ctx, id, isActive, actorID); err != nil {
		return apperror.Internal("")
	}
	return nil
}

func (s *ServiceImpl) Delete(ctx context.Context, id int64) error {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return apperror.NotFound("Buku tidak ditemukan")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return apperror.Internal("")
	}
	// DB first, then delete the files — reverse of Update's order: there's no
	// "new record without a cover" risk here if file deletion fails partway.
	if existing.CoverImage != nil {
		_ = s.upload.DeleteFile(*existing.CoverImage)
	}
	if existing.BookPdf != nil {
		_ = s.upload.DeleteFile(*existing.BookPdf)
	}
	// Best-effort: ms_comment has no FK to ms_catalog_book, so comments aren't
	// cascaded by the database — clean them up explicitly, same as
	// article/news/event. A failure here does not roll back the book delete.
	_ = s.comment.DeleteByContent(ctx, "catalogBook", id)
	return nil
}

func (s *ServiceImpl) Like(ctx context.Context, id int64) (int, error) {
	count, err := s.repo.IncrementFavorite(ctx, id)
	if err != nil {
		if errors.Is(err, catalogbook_repository.ErrNotFound) {
			return 0, apperror.NotFound("Buku tidak ditemukan")
		}
		return 0, apperror.Internal("")
	}
	return count, nil
}

func (s *ServiceImpl) fromRequest(req catalogbook_dto.Request) catalogbook_model.CatalogBook {
	return catalogbook_model.CatalogBook{
		ISBN:               ptr.Str(req.ISBN),
		BookTitle:          req.BookTitle,
		AuthorName:         req.AuthorName,
		AuthorTypeID:       req.AuthorTypeID,
		PublisherName:      req.PublisherName,
		BookCategoryID:     req.BookCategoryID,
		LanguageID:         req.LanguageID,
		AvailabilityTypeID: req.AvailabilityTypeID,
		BookPdf:            ptr.Str(req.BookPdf),
		Year:               req.Year,
		Pages:              req.Pages,
		Description:        req.Description,
		Synopsis:           ptr.Str(req.Synopsis),
		Edition:            ptr.Str(req.Edition),
		CoverImage:         ptr.Str(req.CoverImage),
		Tags:               ptr.Str(req.Tags),
		MetaKeywords:       ptr.Str(req.MetaKeywords),
		MetaDescription:    ptr.Str(req.MetaDescription),
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
