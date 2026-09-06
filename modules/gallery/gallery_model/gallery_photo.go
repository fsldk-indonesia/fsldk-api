package gallery_model

import "time"

// GalleryPhoto represents an associated photo entry in map_gallery_photo.
type GalleryPhoto struct {
	PhotoID      int64     `gorm:"column:photoID;primaryKey;autoIncrement"`
	GalleryID    int64     `gorm:"column:galleryID;not null;index"`
	ImagePath    string    `gorm:"column:imagePath"`
	SortOrder    int       `gorm:"column:sortOrder;default:0"`
	Caption      *string   `gorm:"column:caption"`
	UploadedDate time.Time `gorm:"column:uploadedDate;autoCreateTime"`
	UploadedBy   *int64    `gorm:"column:uploadedBy"`
}

// TableName overrides the default table name for GORM.
func (GalleryPhoto) TableName() string { return "map_gallery_photo" }
