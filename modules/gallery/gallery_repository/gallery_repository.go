package gallery_repository

import (
	"context"
	"errors"

	"fsldk-api/modules/gallery/gallery_dto"
	"fsldk-api/modules/gallery/gallery_model"
)

var (
	// ErrNotFound indicates the requested gallery record does not exist.
	ErrNotFound = errors.New("gallery not found")

	// ErrPhotoNotFound indicates the requested photo record does not exist.
	ErrPhotoNotFound = errors.New("photo not found")
)

// Repository defines database operations for galleries and gallery photos.
type Repository interface {
	List(ctx context.Context, f gallery_dto.Filter) ([]gallery_model.Gallery, int64, error)
	FindByID(ctx context.Context, id int64) (gallery_model.Gallery, error)
	Create(ctx context.Context, g gallery_model.Gallery, photos []gallery_model.GalleryPhoto, authorID int64) (int64, error)
	Update(ctx context.Context, id int64, g gallery_model.Gallery, updatedBy int64) error
	Delete(ctx context.Context, id int64) error

	ListPhotos(ctx context.Context, galleryID int64, limit, offset int) ([]gallery_model.GalleryPhoto, int64, error)
	FindAllPhotos(ctx context.Context, galleryID int64) ([]gallery_model.GalleryPhoto, error)
	FindPhotoByID(ctx context.Context, photoID, galleryID int64) (gallery_model.GalleryPhoto, error)
	AddPhoto(ctx context.Context, photo gallery_model.GalleryPhoto, authorID int64) (int64, error)
	UpdatePhoto(ctx context.Context, photoID, galleryID int64, caption *string, sortOrder *int) error
	DeletePhoto(ctx context.Context, photoID, galleryID int64) error
	ReorderPhotos(ctx context.Context, galleryID int64, orderedIDs []int64) error
	IncrementTotalPhotos(ctx context.Context, galleryID int64, delta int) error
}
