package gallery_service

import (
	"context"
	"math"
	"regexp"
	"strings"
	"time"

	"fsldk-api/base/apperror"
	"fsldk-api/modules/gallery/gallery_dto"
	"fsldk-api/modules/gallery/gallery_model"
	"fsldk-api/modules/gallery/gallery_repository"
)

// FileDeleter provides a contract for removing physical files from disk.
type FileDeleter interface {
	DeleteFile(publicURL string) error
}

var youtubeIDRegex = regexp.MustCompile(`(?i)(?:youtube\.com\/(?:watch\?v=|embed\/|v\/|shorts\/)|youtu\.be\/)([a-zA-Z0-9_-]{11})`)

// extractYouTubeID extracts a clean 11-character YouTube video ID from a URL or raw string.
func extractYouTubeID(input *string) *string {
	if input == nil {
		return nil
	}
	s := strings.TrimSpace(*input)
	if s == "" {
		return nil
	}
	if matches := youtubeIDRegex.FindStringSubmatch(s); len(matches) > 1 {
		return &matches[1]
	}
	if !strings.ContainsAny(s, "/.?&=") && len(s) >= 8 && len(s) <= 20 {
		return &s
	}
	return &s
}

// parseEventDate parses input date strings in various standard formats.
func parseEventDate(input *string) *time.Time {
	if input == nil || strings.TrimSpace(*input) == "" {
		return nil
	}
	s := strings.TrimSpace(*input)
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04",
		"2006-01-02",
		time.RFC3339,
		time.RFC3339Nano,
	}
	for _, f := range formats {
		if t, err := time.ParseInLocation(f, s, time.Local); err == nil {
			return &t
		}
		if t, err := time.Parse(f, s); err == nil {
			return &t
		}
	}
	if len(s) >= 10 {
		if t, err := time.ParseInLocation("2006-01-02", s[:10], time.Local); err == nil {
			return &t
		}
	}
	return nil
}

type serviceImpl struct {
	repo   gallery_repository.Repository
	upload FileDeleter
}

// NewService creates a new gallery service instance.
func NewService(repo gallery_repository.Repository, upload FileDeleter) Service {
	return &serviceImpl{
		repo:   repo,
		upload: upload,
	}
}

func (s *serviceImpl) ListPublic(ctx context.Context, page, limit int, sort string) ([]gallery_dto.GalleryListItem, int64, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 9
	}
	offset := (page - 1) * limit

	sortOrder := "DESC"
	if strings.ToLower(sort) == "oldest" {
		sortOrder = "ASC"
	}

	filter := gallery_dto.Filter{
		SortBy:    "COALESCE(eventDate, createdDate)",
		SortOrder: sortOrder,
		Limit:     limit,
		Offset:    offset,
	}

	items, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, 0, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages < 1 {
		totalPages = 1
	}

	list := make([]gallery_dto.GalleryListItem, len(items))
	for i, it := range items {
		list[i] = gallery_dto.GalleryListItem{
			GalleryID:      it.GalleryID,
			EventName:      it.EventName,
			EventTheme:     it.EventTheme,
			EventDate:      it.EventDate,
			CoverImage:     it.CoverImage,
			YoutubeVideoID: it.YoutubeVideoID,
			TotalPhotos:    it.TotalPhotos,
			CreatedDate:    it.CreatedDate,
		}
	}

	return list, total, totalPages, nil
}

func (s *serviceImpl) GetPublic(ctx context.Context, id int64) (gallery_dto.GalleryDetailResponse, error) {
	item, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if err == gallery_repository.ErrNotFound {
			return gallery_dto.GalleryDetailResponse{}, apperror.NotFound("Galeri tidak ditemukan")
		}
		return gallery_dto.GalleryDetailResponse{}, err
	}

	return gallery_dto.GalleryDetailResponse{
		GalleryID:        item.GalleryID,
		EventName:        item.EventName,
		EventTheme:       item.EventTheme,
		EventDate:        item.EventDate,
		EventDescription: item.EventDescription,
		CoverImage:       item.CoverImage,
		YoutubeVideoID:   item.YoutubeVideoID,
		DocumentLink:     item.DocumentLink,
		TotalPhotos:      item.TotalPhotos,
		CreatedDate:      item.CreatedDate,
	}, nil
}

func (s *serviceImpl) ListPhotosPublic(ctx context.Context, galleryID int64, page, limit int) (gallery_dto.PhotoPageResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 12
	}
	offset := (page - 1) * limit

	// Ensure gallery exists first
	if _, err := s.repo.FindByID(ctx, galleryID); err != nil {
		if err == gallery_repository.ErrNotFound {
			return gallery_dto.PhotoPageResponse{}, apperror.NotFound("Galeri tidak ditemukan")
		}
		return gallery_dto.PhotoPageResponse{}, err
	}

	photos, total, err := s.repo.ListPhotos(ctx, galleryID, limit, offset)
	if err != nil {
		return gallery_dto.PhotoPageResponse{}, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages < 1 {
		totalPages = 1
	}

	data := make([]gallery_dto.PhotoResponse, len(photos))
	for i, p := range photos {
		data[i] = gallery_dto.PhotoResponse{
			PhotoID:      p.PhotoID,
			GalleryID:    p.GalleryID,
			ImagePath:    p.ImagePath,
			Caption:      p.Caption,
			SortOrder:    p.SortOrder,
			UploadedDate: p.UploadedDate,
		}
	}

	return gallery_dto.PhotoPageResponse{
		GalleryID:  galleryID,
		Data:       data,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func (s *serviceImpl) ListCMS(ctx context.Context, f gallery_dto.Filter) ([]gallery_dto.GalleryListItem, int64, error) {
	items, total, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, 0, err
	}

	list := make([]gallery_dto.GalleryListItem, len(items))
	for i, it := range items {
		list[i] = gallery_dto.GalleryListItem{
			GalleryID:      it.GalleryID,
			EventName:      it.EventName,
			EventTheme:     it.EventTheme,
			EventDate:      it.EventDate,
			CoverImage:     it.CoverImage,
			YoutubeVideoID: it.YoutubeVideoID,
			TotalPhotos:    it.TotalPhotos,
			CreatedDate:    it.CreatedDate,
		}
	}
	return list, total, nil
}

func (s *serviceImpl) GetCMS(ctx context.Context, id int64) (gallery_dto.GalleryDetailResponse, error) {
	return s.GetPublic(ctx, id)
}

func (s *serviceImpl) Create(ctx context.Context, req gallery_dto.CreateRequest, authorID int64) (int64, error) {
	videoID := extractYouTubeID(req.YoutubeVideoID)
	eventDate := parseEventDate(req.EventDate)

	galleryModel := gallery_model.Gallery{
		EventName:        req.EventName,
		EventTheme:       req.EventTheme,
		EventDate:        eventDate,
		EventDescription: req.EventDescription,
		CoverImage:       req.CoverImage,
		YoutubeVideoID:   videoID,
		DocumentLink:     req.DocumentLink,
	}

	photos := make([]gallery_model.GalleryPhoto, len(req.Photos))
	for i, p := range req.Photos {
		sortOrder := p.SortOrder
		if sortOrder == 0 && i > 0 {
			sortOrder = i
		}
		photos[i] = gallery_model.GalleryPhoto{
			ImagePath: p.ImagePath,
			Caption:   p.Caption,
			SortOrder: sortOrder,
		}
	}

	return s.repo.Create(ctx, galleryModel, photos, authorID)
}

func (s *serviceImpl) Update(ctx context.Context, id int64, req gallery_dto.UpdateRequest, updatedBy int64) error {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if err == gallery_repository.ErrNotFound {
			return apperror.NotFound("Galeri tidak ditemukan")
		}
		return err
	}

	videoID := extractYouTubeID(req.YoutubeVideoID)
	eventDate := parseEventDate(req.EventDate)

	galleryModel := gallery_model.Gallery{
		EventName:        req.EventName,
		EventTheme:       req.EventTheme,
		EventDate:        eventDate,
		EventDescription: req.EventDescription,
		CoverImage:       req.CoverImage,
		YoutubeVideoID:   videoID,
		DocumentLink:     req.DocumentLink,
	}

	// If cover image changed, remove old cover file
	if existing.CoverImage != "" && existing.CoverImage != req.CoverImage && s.upload != nil {
		_ = s.upload.DeleteFile(existing.CoverImage)
	}

	return s.repo.Update(ctx, id, galleryModel, updatedBy)
}

func (s *serviceImpl) Delete(ctx context.Context, id int64) error {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if err == gallery_repository.ErrNotFound {
			return apperror.NotFound("Galeri tidak ditemukan")
		}
		return err
	}

	// Fetch all associated photos to clean up disk files before cascading DB delete
	photos, err := s.repo.FindAllPhotos(ctx, id)
	if err == nil && s.upload != nil {
		for _, p := range photos {
			if p.ImagePath != "" {
				_ = s.upload.DeleteFile(p.ImagePath)
			}
		}
		if existing.CoverImage != "" {
			_ = s.upload.DeleteFile(existing.CoverImage)
		}
	}

	return s.repo.Delete(ctx, id)
}

func (s *serviceImpl) ListPhotosCMS(ctx context.Context, galleryID int64, page, limit int) (gallery_dto.PhotoPageResponse, error) {
	return s.ListPhotosPublic(ctx, galleryID, page, limit)
}

func (s *serviceImpl) AddPhoto(ctx context.Context, galleryID int64, req gallery_dto.AddPhotoRequest, authorID int64) (gallery_dto.PhotoResponse, error) {
	if _, err := s.repo.FindByID(ctx, galleryID); err != nil {
		if err == gallery_repository.ErrNotFound {
			return gallery_dto.PhotoResponse{}, apperror.NotFound("Galeri tidak ditemukan")
		}
		return gallery_dto.PhotoResponse{}, err
	}

	photoModel := gallery_model.GalleryPhoto{
		GalleryID: galleryID,
		ImagePath: req.ImagePath,
		Caption:   req.Caption,
		SortOrder: req.SortOrder,
	}

	photoID, err := s.repo.AddPhoto(ctx, photoModel, authorID)
	if err != nil {
		return gallery_dto.PhotoResponse{}, err
	}

	photo, err := s.repo.FindPhotoByID(ctx, photoID, galleryID)
	if err != nil {
		return gallery_dto.PhotoResponse{}, err
	}

	return gallery_dto.PhotoResponse{
		PhotoID:      photo.PhotoID,
		GalleryID:    photo.GalleryID,
		ImagePath:    photo.ImagePath,
		Caption:      photo.Caption,
		SortOrder:    photo.SortOrder,
		UploadedDate: photo.UploadedDate,
	}, nil
}

func (s *serviceImpl) UpdatePhoto(ctx context.Context, galleryID, photoID int64, req gallery_dto.UpdatePhotoRequest) error {
	err := s.repo.UpdatePhoto(ctx, photoID, galleryID, req.Caption, req.SortOrder)
	if err != nil {
		if err == gallery_repository.ErrPhotoNotFound {
			return apperror.NotFound("Foto tidak ditemukan")
		}
		return err
	}
	return nil
}

func (s *serviceImpl) DeletePhoto(ctx context.Context, galleryID, photoID int64) error {
	photo, err := s.repo.FindPhotoByID(ctx, photoID, galleryID)
	if err != nil {
		if err == gallery_repository.ErrPhotoNotFound {
			return apperror.NotFound("Foto tidak ditemukan")
		}
		return err
	}

	// Delete physical file
	if photo.ImagePath != "" && s.upload != nil {
		_ = s.upload.DeleteFile(photo.ImagePath)
	}

	return s.repo.DeletePhoto(ctx, photoID, galleryID)
}

func (s *serviceImpl) ReorderPhotos(ctx context.Context, galleryID int64, req gallery_dto.ReorderPhotosRequest) error {
	return s.repo.ReorderPhotos(ctx, galleryID, req.Order)
}
