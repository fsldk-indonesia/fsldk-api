// Package financeformat_dto holds financeformat request/response DTOs. Pure data, no methods.
package financeformat_dto

import "fsldk-api/modules/financeformat/financeformat_model"

// Request is the body for creating/updating a finance format.
type Request struct {
	FileName     string `json:"fileName" validate:"required,min=3,max=255"`
	FileURL      string `json:"fileURL" validate:"required,max=500"` // URL from POST /uploads/document, must end in .xlsx (validated in service)
	FormatTypeID int64  `json:"formatTypeID" validate:"required"`
}

// PublishRequest is the body for toggling public visibility (financeformat.publish).
type PublishRequest struct {
	IsActive bool `json:"isActive"`
}

// Filter holds finance format list filter parameters (repository & service).
type Filter struct {
	Search       string // LIKE against fileName
	FormatTypeID int64  // 0 = all categories
	DateFrom     string // "2006-01-02", optional — matched against DATE(createdDate)
	DateTo       string
	ActiveOnly   bool // true for the public endpoint, false for CMS
	Limit        int
	Offset       int
	OrderBy      string
}

// PublicListResponse is the combined payload of GET /public/finance-formats:
// the 9 fixed categories plus every active file, so the frontend can render
// all categories (including empty ones) from one request.
type PublicListResponse struct {
	FormatTypes []financeformat_model.FormatType    `json:"formatTypes"`
	Formats     []financeformat_model.FinanceFormat `json:"formats"`
	CpName      string                              `json:"cpName"`
	CpPhone     string                              `json:"cpPhone"`
}
