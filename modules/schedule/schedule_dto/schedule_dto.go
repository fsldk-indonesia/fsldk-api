// Package schedule_dto holds schedule request/response DTOs. Pure data, no methods.
package schedule_dto

// Request is the body for creating/updating a schedule (see tech spec §4).
// Cross-field rules (endDate >= startDate, startTime required unless isAllDay,
// endTime > startTime for single-day activities, 366-day cap) are checked in
// the service, not by tags.
type Request struct {
	Title         string `json:"title" validate:"required,min=3,max=150"`
	Category      string `json:"category" validate:"required,oneof=kajian rapat daurah aksi kaderisasi keputrian lomba libur lainnya"`
	Description   string `json:"description" validate:"max=2000"`
	StartDate     string `json:"startDate" validate:"required"` // "YYYY-MM-DD"
	EndDate       string `json:"endDate"`                       // "YYYY-MM-DD", optional
	IsAllDay      bool   `json:"isAllDay"`
	StartTime     string `json:"startTime"` // "HH:mm", required when isAllDay is false
	EndTime       string `json:"endTime"`   // "HH:mm", optional
	Location      string `json:"location" validate:"max=200"`
	Organizer     string `json:"organizer" validate:"max=150"`
	ContactPerson string `json:"contactPerson" validate:"max=100"`
	URL           string `json:"url" validate:"omitempty,url,max=300"`
}

// PublishRequest is the body for toggling public visibility (schedule.publish).
type PublishRequest struct {
	IsActive bool `json:"isActive"`
}

// Filter holds schedule list filter parameters (repository & service).
type Filter struct {
	Search     string // LIKE against title
	Category   string // "" = all
	Month      int    // 0 = all; matched against MONTH(startDate)
	Year       int    // 0 = all; matched against YEAR(startDate)
	DateFrom   string // "YYYY-MM-DD", optional — lower bound of the overlap window
	DateTo     string // "YYYY-MM-DD", optional — upper bound of the overlap window
	ActiveOnly bool   // true for the public endpoint, false for CMS
	Limit      int
	Offset     int
	OrderBy    string
}

// Response is the schedule shape returned by the API: dates as "YYYY-MM-DD",
// times as "HH:mm", createdDate/updatedDate as "YYYY-MM-DD HH:mm:ss".
type Response struct {
	ScheduleID    int64   `json:"scheduleID"`
	Title         string  `json:"title"`
	Category      string  `json:"category"`
	Description   *string `json:"description"`
	StartDate     string  `json:"startDate"`
	EndDate       *string `json:"endDate"`
	IsAllDay      bool    `json:"isAllDay"`
	StartTime     *string `json:"startTime"`
	EndTime       *string `json:"endTime"`
	Location      *string `json:"location"`
	Organizer     *string `json:"organizer"`
	ContactPerson *string `json:"contactPerson"`
	URL           *string `json:"url"`
	IsActive      bool    `json:"isActive"`
	CreatedDate   string  `json:"createdDate"`
	UpdatedDate   *string `json:"updatedDate"`
}
