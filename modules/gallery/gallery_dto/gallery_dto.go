package gallery_dto

import "time"

// CreatePhotoItem represents a photo item in the gallery creation payload.
type CreatePhotoItem struct {
	ImagePath string  `json:"imagePath" validate:"required"`
	Caption   *string `json:"caption" validate:"omitempty,max=255"`
	SortOrder int     `json:"sortOrder"`
}

// CreateRequest represents the payload to create a new gallery.
type CreateRequest struct {
	EventName        string            `json:"eventName" validate:"required,max=255"`
	EventTheme       string            `json:"eventTheme" validate:"required,max=255"`
	EventDate        *string           `json:"eventDate" validate:"omitempty"`
	EventDescription string            `json:"eventDescription" validate:"required"`
	CoverImage       string            `json:"coverImage" validate:"required"`
	YoutubeVideoID   *string           `json:"youtubeVideoID" validate:"omitempty"`
	DocumentLink     *string           `json:"documentLink" validate:"omitempty,max=500"`
	Photos           []CreatePhotoItem `json:"photos" validate:"omitempty,dive"`
}

// UpdateRequest represents the payload to update gallery metadata.
type UpdateRequest struct {
	EventName        string  `json:"eventName" validate:"required,max=255"`
	EventTheme       string  `json:"eventTheme" validate:"required,max=255"`
	EventDate        *string `json:"eventDate" validate:"omitempty"`
	EventDescription string  `json:"eventDescription" validate:"required"`
	CoverImage       string  `json:"coverImage" validate:"required"`
	YoutubeVideoID   *string `json:"youtubeVideoID" validate:"omitempty"`
	DocumentLink     *string `json:"documentLink" validate:"omitempty,max=500"`
}

// AddPhotoRequest represents the payload to add a single photo to an existing gallery.
type AddPhotoRequest struct {
	ImagePath string  `json:"imagePath" validate:"required"`
	Caption   *string `json:"caption" validate:"omitempty,max=255"`
	SortOrder int     `json:"sortOrder"`
}

// UpdatePhotoRequest represents the payload to update a photo's caption and/or sort order.
type UpdatePhotoRequest struct {
	Caption   *string `json:"caption" validate:"omitempty,max=255"`
	SortOrder *int    `json:"sortOrder" validate:"omitempty,min=0"`
}

// ReorderPhotosRequest represents the payload to reorder photos by their IDs.
type ReorderPhotosRequest struct {
	Order []int64 `json:"order" validate:"required,min=1"`
}

// Filter holds query parameters for filtering and paginating galleries.
type Filter struct {
	Search     string
	EventName  string
	EventTheme string
	SortBy     string
	SortOrder  string
	Limit      int
	Offset     int
}

// GalleryListItem represents a compact gallery entry for list endpoints.
type GalleryListItem struct {
	GalleryID      int64      `json:"galleryID"`
	EventName      string     `json:"eventName"`
	EventTheme     string     `json:"eventTheme"`
	EventDate      *time.Time `json:"eventDate"`
	CoverImage     string     `json:"coverImage"`
	YoutubeVideoID *string    `json:"youtubeVideoID"`
	TotalPhotos    int        `json:"totalPhotos"`
	CreatedDate    time.Time  `json:"createdDate"`
}

// GalleryDetailResponse represents the full metadata of a gallery.
type GalleryDetailResponse struct {
	GalleryID        int64      `json:"galleryID"`
	EventName        string     `json:"eventName"`
	EventTheme       string     `json:"eventTheme"`
	EventDate        *time.Time `json:"eventDate"`
	EventDescription string     `json:"eventDescription"`
	CoverImage       string     `json:"coverImage"`
	YoutubeVideoID   *string    `json:"youtubeVideoID"`
	DocumentLink     *string    `json:"documentLink"`
	TotalPhotos      int        `json:"totalPhotos"`
	CreatedDate      time.Time  `json:"createdDate"`
}

// PhotoResponse represents an individual photo item response.
type PhotoResponse struct {
	PhotoID      int64     `json:"photoID"`
	GalleryID    int64     `json:"galleryID"`
	ImagePath    string    `json:"imagePath"`
	Caption      *string   `json:"caption"`
	SortOrder    int       `json:"sortOrder"`
	UploadedDate time.Time `json:"uploadedDate"`
}

// PhotoPageResponse represents paginated photos response for a gallery.
type PhotoPageResponse struct {
	GalleryID  int64           `json:"galleryID"`
	Data       []PhotoResponse `json:"data"`
	Page       int             `json:"page"`
	Limit      int             `json:"limit"`
	Total      int64           `json:"total"`
	TotalPages int             `json:"totalPages"`
}
