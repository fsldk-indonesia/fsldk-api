package gallery_repository

import (
	"context"
	"errors"
	"strings"

	"fsldk-api/modules/gallery/gallery_dto"
	"fsldk-api/modules/gallery/gallery_model"

	"gorm.io/gorm"
)

type repositoryImpl struct {
	db *gorm.DB
}

// NewRepository creates a new gallery repository instance.
func NewRepository(db *gorm.DB) Repository {
	return &repositoryImpl{db: db}
}

func (r *repositoryImpl) List(ctx context.Context, f gallery_dto.Filter) ([]gallery_model.Gallery, int64, error) {
	var items []gallery_model.Gallery
	var total int64

	q := r.db.WithContext(ctx).Model(&gallery_model.Gallery{})

	if f.Search != "" {
		pattern := "%" + strings.ToLower(f.Search) + "%"
		q = q.Where("LOWER(eventName) LIKE ? OR LOWER(eventTheme) LIKE ?", pattern, pattern)
	}
	if f.EventName != "" {
		q = q.Where("LOWER(eventName) LIKE ?", "%"+strings.ToLower(f.EventName)+"%")
	}
	if f.EventTheme != "" {
		q = q.Where("LOWER(eventTheme) LIKE ?", "%"+strings.ToLower(f.EventTheme)+"%")
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sortBy := "COALESCE(eventDate, createdDate)"
	switch f.SortBy {
	case "eventName", "eventTheme", "totalPhotos", "createdDate", "eventDate":
		sortBy = f.SortBy
	}

	sortOrder := "DESC"
	if strings.ToUpper(f.SortOrder) == "ASC" {
		sortOrder = "ASC"
	}

	q = q.Order(sortBy + " " + sortOrder)

	if f.Limit > 0 {
		q = q.Limit(f.Limit).Offset(f.Offset)
	}

	if err := q.Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *repositoryImpl) FindByID(ctx context.Context, id int64) (gallery_model.Gallery, error) {
	var item gallery_model.Gallery
	err := r.db.WithContext(ctx).Where("galleryID = ?", id).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return item, ErrNotFound
		}
		return item, err
	}
	return item, nil
}

func (r *repositoryImpl) Create(ctx context.Context, g gallery_model.Gallery, photos []gallery_model.GalleryPhoto, authorID int64) (int64, error) {
	var createdID int64

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		g.CreatedBy = &authorID
		g.TotalPhotos = len(photos)

		if err := tx.Create(&g).Error; err != nil {
			return err
		}
		createdID = g.GalleryID

		for i := range photos {
			photos[i].GalleryID = createdID
			photos[i].UploadedBy = &authorID
			if err := tx.Create(&photos[i]).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return 0, err
	}
	return createdID, nil
}

func (r *repositoryImpl) Update(ctx context.Context, id int64, g gallery_model.Gallery, updatedBy int64) error {
	updates := map[string]interface{}{
		"eventName":        g.EventName,
		"eventTheme":       g.EventTheme,
		"eventDate":        g.EventDate,
		"eventDescription": g.EventDescription,
		"coverImage":       g.CoverImage,
		"youtubeVideoID":   g.YoutubeVideoID,
		"documentLink":     g.DocumentLink,
		"updatedBy":        updatedBy,
	}

	res := r.db.WithContext(ctx).Model(&gallery_model.Gallery{}).Where("galleryID = ?", id).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *repositoryImpl) Delete(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Where("galleryID = ?", id).Delete(&gallery_model.Gallery{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *repositoryImpl) ListPhotos(ctx context.Context, galleryID int64, limit, offset int) ([]gallery_model.GalleryPhoto, int64, error) {
	var photos []gallery_model.GalleryPhoto
	var total int64

	q := r.db.WithContext(ctx).Model(&gallery_model.GalleryPhoto{}).Where("galleryID = ?", galleryID)

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	q = q.Order("sortOrder ASC, photoID ASC")
	if limit > 0 {
		q = q.Limit(limit).Offset(offset)
	}

	if err := q.Find(&photos).Error; err != nil {
		return nil, 0, err
	}

	return photos, total, nil
}

func (r *repositoryImpl) FindAllPhotos(ctx context.Context, galleryID int64) ([]gallery_model.GalleryPhoto, error) {
	var photos []gallery_model.GalleryPhoto
	err := r.db.WithContext(ctx).Where("galleryID = ?", galleryID).Order("sortOrder ASC, photoID ASC").Find(&photos).Error
	return photos, err
}

func (r *repositoryImpl) FindPhotoByID(ctx context.Context, photoID, galleryID int64) (gallery_model.GalleryPhoto, error) {
	var photo gallery_model.GalleryPhoto
	err := r.db.WithContext(ctx).Where("photoID = ? AND galleryID = ?", photoID, galleryID).First(&photo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return photo, ErrPhotoNotFound
		}
		return photo, err
	}
	return photo, nil
}

func (r *repositoryImpl) AddPhoto(ctx context.Context, photo gallery_model.GalleryPhoto, authorID int64) (int64, error) {
	var photoID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		photo.UploadedBy = &authorID
		if err := tx.Create(&photo).Error; err != nil {
			return err
		}
		photoID = photo.PhotoID

		// Increment denormalized totalPhotos count
		return tx.Model(&gallery_model.Gallery{}).
			Where("galleryID = ?", photo.GalleryID).
			UpdateColumn("totalPhotos", gorm.Expr("totalPhotos + 1")).Error
	})

	if err != nil {
		return 0, err
	}
	return photoID, nil
}

func (r *repositoryImpl) UpdatePhoto(ctx context.Context, photoID, galleryID int64, caption *string, sortOrder *int) error {
	updates := map[string]interface{}{}
	if caption != nil {
		updates["caption"] = *caption
	}
	if sortOrder != nil {
		updates["sortOrder"] = *sortOrder
	}
	if len(updates) == 0 {
		return nil
	}

	res := r.db.WithContext(ctx).Model(&gallery_model.GalleryPhoto{}).
		Where("photoID = ? AND galleryID = ?", photoID, galleryID).
		Updates(updates)

	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrPhotoNotFound
	}
	return nil
}

func (r *repositoryImpl) DeletePhoto(ctx context.Context, photoID, galleryID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("photoID = ? AND galleryID = ?", photoID, galleryID).Delete(&gallery_model.GalleryPhoto{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrPhotoNotFound
		}

		// Decrement denormalized totalPhotos count (keep >= 0)
		return tx.Model(&gallery_model.Gallery{}).
			Where("galleryID = ? AND totalPhotos > 0", galleryID).
			UpdateColumn("totalPhotos", gorm.Expr("totalPhotos - 1")).Error
	})
}

func (r *repositoryImpl) ReorderPhotos(ctx context.Context, galleryID int64, orderedIDs []int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for idx, photoID := range orderedIDs {
			if err := tx.Model(&gallery_model.GalleryPhoto{}).
				Where("photoID = ? AND galleryID = ?", photoID, galleryID).
				Update("sortOrder", idx).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *repositoryImpl) IncrementTotalPhotos(ctx context.Context, galleryID int64, delta int) error {
	return r.db.WithContext(ctx).Model(&gallery_model.Gallery{}).
		Where("galleryID = ?", galleryID).
		UpdateColumn("totalPhotos", gorm.Expr("totalPhotos + ?", delta)).Error
}
