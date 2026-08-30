// Package catalogbook_dto holds catalogbook request/response DTOs. Pure data, no methods.
package catalogbook_dto

// Request is the body for creating/updating a book.
type Request struct {
	ISBN               string `json:"isbn" validate:"max=50"`
	BookTitle          string `json:"bookTitle" validate:"required,min=2,max=255"`
	AuthorName         string `json:"authorName" validate:"required,min=2,max=255"`
	AuthorTypeID       int64  `json:"authorTypeID" validate:"required"`
	PublisherName      string `json:"publisherName" validate:"required,min=2,max=255"`
	BookCategoryID     int64  `json:"bookCategoryID" validate:"required"`
	LanguageID         int64  `json:"languageID" validate:"required"`
	AvailabilityTypeID int64  `json:"availabilityTypeID" validate:"required"`
	BookPdf            string `json:"bookPdf" validate:"max=500"` // URL returned by POST /uploads/document
	Year               string `json:"year" validate:"required,len=4"`
	Pages              int    `json:"pages" validate:"required,min=1"`
	Description        string `json:"description" validate:"required"`
	Synopsis           string `json:"synopsis"`
	Edition            string `json:"edition" validate:"max=100"`
	CoverImage         string `json:"coverImage" validate:"max=500"` // URL returned by POST /uploads/image
	Tags               string `json:"tags"`
	MetaKeywords       string `json:"metaKeywords"`
	MetaDescription    string `json:"metaDescription"`
}

// PublishRequest is the body for toggling public visibility (catalogbook.publish).
type PublishRequest struct {
	IsActive bool `json:"isActive"`
}

// LikeResponse is the response body of the like endpoint.
type LikeResponse struct {
	FavoriteCount int `json:"favoriteCount"`
}

// Filter holds book list filter parameters (repository & service). Slice
// fields accept multiple values (?bookCategoryID=1&bookCategoryID=2 or
// ?bookCategoryID=1,2 — parsed in the handler).
type Filter struct {
	Search              string
	BookCategoryIDs     []int64
	AuthorTypeIDs       []int64
	AvailabilityTypeIDs []int64
	LanguageIDs         []int64
	Years               []string
	Author              string
	Publisher           string
	ActiveOnly          bool // true for the public endpoint, false for CMS
	Limit               int
	Offset              int
	OrderBy             string
}
