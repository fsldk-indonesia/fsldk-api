package gallery_service

import (
	"context"

	"fsldk-api/modules/gallery/gallery_dto"
)

// Service defines the business logic operations for the gallery module.
type Service interface {
	// Public operations
	ListPublic(ctx context.Context, page, limit int, sort, search string, eventNames []string, years []int) ([]gallery_dto.GalleryListItem, int64, int, error)
	// FilterOptionsPublic returns the distinct years/event names for the
	// public filter dropdowns (Tahun Kegiatan / Nama Kegiatan).
	FilterOptionsPublic(ctx context.Context) (gallery_dto.FilterOptionsResponse, error)
	GetPublic(ctx context.Context, id int64) (gallery_dto.GalleryDetailResponse, error)
	// GetPublicBySlug cari galeri lewat gallerySlug; kalau slug-nya murni
	// digit, fallback ke galleryID (backward-compat link lama /galeri/<id>).
	GetPublicBySlug(ctx context.Context, slug string) (gallery_dto.GalleryDetailResponse, error)
	// ListPhotosPublic menerima slug (sama fallback ID seperti GetPublicBySlug).
	ListPhotosPublic(ctx context.Context, slug string, page, limit int) (gallery_dto.PhotoPageResponse, error)

	// CMS operations
	ListCMS(ctx context.Context, f gallery_dto.Filter) ([]gallery_dto.GalleryListItem, int64, error)
	GetCMS(ctx context.Context, id int64) (gallery_dto.GalleryDetailResponse, error)
	Create(ctx context.Context, req gallery_dto.CreateRequest, authorID int64) (int64, error)
	Update(ctx context.Context, id int64, req gallery_dto.UpdateRequest, updatedBy int64) error
	Delete(ctx context.Context, id int64) error
	BulkDelete(ctx context.Context, ids []int64) error

	// CMS photo management operations
	ListPhotosCMS(ctx context.Context, galleryID int64, page, limit int) (gallery_dto.PhotoPageResponse, error)
	AddPhoto(ctx context.Context, galleryID int64, req gallery_dto.AddPhotoRequest, authorID int64) (gallery_dto.PhotoResponse, error)
	UpdatePhoto(ctx context.Context, galleryID, photoID int64, req gallery_dto.UpdatePhotoRequest) error
	DeletePhoto(ctx context.Context, galleryID, photoID int64) error
	ReorderPhotos(ctx context.Context, galleryID int64, req gallery_dto.ReorderPhotosRequest) error
}
