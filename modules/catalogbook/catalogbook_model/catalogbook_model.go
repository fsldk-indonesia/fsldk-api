// Package catalogbook_model holds catalogbook module entities. Pure data, no methods.
package catalogbook_model

import "time"

// CatalogBook represents one ms_catalog_book row.
type CatalogBook struct {
	BookID             int64     `gorm:"column:bookID;primaryKey" json:"bookID"`
	BookSlug           string    `gorm:"column:bookSlug" json:"bookSlug"`
	ISBN               *string   `gorm:"column:isbn" json:"isbn"`
	BookTitle          string    `gorm:"column:bookTitle" json:"bookTitle"`
	AuthorName         string    `gorm:"column:authorName" json:"authorName"`
	AuthorTypeID       int64     `gorm:"column:authorTypeID" json:"authorTypeID"`
	AuthorTypeName     string    `gorm:"column:authorTypeName;->" json:"authorTypeName"`
	PublisherName      string    `gorm:"column:publisherName" json:"publisherName"`
	BookCategoryID     int64     `gorm:"column:bookCategoryID" json:"bookCategoryID"`
	BookCategoryName   string    `gorm:"column:bookCategoryName;->" json:"bookCategoryName"`
	LanguageID         int64     `gorm:"column:languageID" json:"languageID"`
	LanguageName       string    `gorm:"column:languageName;->" json:"languageName"`
	AvailabilityTypeID int64     `gorm:"column:availabilityTypeID" json:"availabilityTypeID"`
	AvailabilityName   string    `gorm:"column:availabilityTypeName;->" json:"availabilityTypeName"`
	BookPdf            *string   `gorm:"column:bookPdf" json:"bookPdf"`
	Year               string    `gorm:"column:year" json:"year"`
	Pages              int       `gorm:"column:pages" json:"pages"`
	Description        string    `gorm:"column:description" json:"description"`
	Synopsis           *string   `gorm:"column:synopsis" json:"synopsis"`
	Edition            *string   `gorm:"column:edition" json:"edition"`
	CoverImage         *string   `gorm:"column:coverImage" json:"coverImage"`
	FavoriteCount      int       `gorm:"column:favoriteCount" json:"favoriteCount"`
	Tags               *string   `gorm:"column:tags" json:"tags"`
	MetaKeywords       *string   `gorm:"column:metaKeywords" json:"metaKeywords"`
	MetaDescription    *string   `gorm:"column:metaDescription" json:"metaDescription"`
	IsActive           bool      `gorm:"column:isActive" json:"isActive"`
	CreatedDate        time.Time `gorm:"column:createdDate" json:"createdDate"`
}

// BookCategory / Language / AuthorType / AvailabilityType are lookup table rows, read publicly.
type BookCategory struct {
	BookCategoryID   int64  `gorm:"column:bookCategoryID;primaryKey" json:"bookCategoryID"`
	BookCategoryName string `gorm:"column:bookCategoryName" json:"bookCategoryName"`
}

type Language struct {
	LanguageID   int64  `gorm:"column:languageID;primaryKey" json:"languageID"`
	LanguageName string `gorm:"column:languageName" json:"languageName"`
}

type AuthorType struct {
	AuthorTypeID   int64  `gorm:"column:authorTypeID;primaryKey" json:"authorTypeID"`
	AuthorTypeName string `gorm:"column:authorTypeName" json:"authorTypeName"`
}

type AvailabilityType struct {
	AvailabilityTypeID   int64  `gorm:"column:availabilityTypeID;primaryKey" json:"availabilityTypeID"`
	AvailabilityTypeName string `gorm:"column:availabilityTypeName" json:"availabilityTypeName"`
}
