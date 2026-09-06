package gallery_model

import "time"

// Gallery represents an event documentation entry in ms_gallery.
type Gallery struct {
	GalleryID        int64          `gorm:"column:galleryID;primaryKey;autoIncrement"`
	EventName        string         `gorm:"column:eventName"`
	EventTheme       string         `gorm:"column:eventTheme"`
	EventDate        *time.Time     `gorm:"column:eventDate"`
	EventDescription string         `gorm:"column:eventDescription"`
	CoverImage       string         `gorm:"column:coverImage"`
	YoutubeVideoID   *string        `gorm:"column:youtubeVideoID"`
	DocumentLink     *string        `gorm:"column:documentLink"`
	TotalPhotos      int            `gorm:"column:totalPhotos;default:0"`
	CreatedDate      time.Time      `gorm:"column:createdDate;autoCreateTime"`
	CreatedBy        *int64         `gorm:"column:createdBy"`
	UpdatedDate      *time.Time     `gorm:"column:updatedDate;autoUpdateTime"`
	UpdatedBy        *int64         `gorm:"column:updatedBy"`

	// Photos relation (loaded on-demand or in details)
	Photos []GalleryPhoto `gorm:"foreignKey:GalleryID;references:GalleryID"`
}

// TableName overrides the default table name for GORM.
func (Gallery) TableName() string { return "ms_gallery" }
