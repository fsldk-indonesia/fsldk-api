package gallery_service

import (
	"context"

	"fsldk-api/modules/gallery/gallery_dto"
)

// Service defines the business logic operations for the gallery module.
type Service interface {
	// Public operations
	ListPublic(ctx context.Context, page, limit int, sort string) ([]gallery_dto.GalleryListItem, int64, int, error)
	GetPublic(ctx context.Context, id int64) (gallery_dto.GalleryDetailResponse, error)
	ListPhotosPublic(ctx context.Context, galleryID int64, page, limit int) (gallery_dto.PhotoPageResponse, error)

	// CMS operations
	ListCMS(ctx context.Context, f gallery_dto.Filter) ([]gallery_dto.GalleryListItem, int64, error)
	GetCMS(ctx context.Context, id int64) (gallery_dto.GalleryDetailResponse, error)
	Create(ctx context.Context, req gallery_dto.CreateRequest, authorID int64) (int64, error)
	Update(ctx context.Context, id int64, req gallery_dto.UpdateRequest, updatedBy int64) error
	Delete(ctx context.Context, id int64) error

	// CMS photo management operations
	ListPhotosCMS(ctx context.Context, galleryID int64, page, limit int) (gallery_dto.PhotoPageResponse, error)
	AddPhoto(ctx context.Context, galleryID int64, req gallery_dto.AddPhotoRequest, authorID int64) (gallery_dto.PhotoResponse, error)
	UpdatePhoto(ctx context.Context, galleryID, photoID int64, req gallery_dto.UpdatePhotoRequest) error
	DeletePhoto(ctx context.Context, galleryID, photoID int64) error
	ReorderPhotos(ctx context.Context, galleryID int64, req gallery_dto.ReorderPhotosRequest) error
}
