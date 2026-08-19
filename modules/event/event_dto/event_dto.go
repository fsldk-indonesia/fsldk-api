// Package event_dto holds request/response DTOs for the event module.
package event_dto

import "time"

// CreateRequest is the body for creating or updating an event.
type CreateRequest struct {
	EventTitle       string  `json:"eventTitle" validate:"required,max=255"`
	EventDivision    string  `json:"eventDivision" validate:"required,max=150"`
	EventContent     string  `json:"eventContent" validate:"required"`
	EventImage       string  `json:"eventImage" validate:"max=255"`
	StartDate        string  `json:"startDate"`
	EndDate          string  `json:"endDate"`
	CloseRegistDate  string  `json:"closeRegistDate"`
	Location         string  `json:"location" validate:"max=255"`
	Place            string  `json:"place" validate:"max=255"`
	LocationLink     string  `json:"locationLink" validate:"max=500"`
	RegistrationLink string  `json:"registrationLink" validate:"max=500"`
	DocumentLink     string  `json:"documentLink" validate:"max=500"`
	PresentationLink string  `json:"presentationLink" validate:"max=500"`
	ContactPerson1   string  `json:"contactPerson1" validate:"max=30"`
	NameCp1          string  `json:"nameCp1" validate:"max=100"`
	ContactPerson2   string  `json:"contactPerson2" validate:"max=30"`
	NameCp2          string  `json:"nameCp2" validate:"max=100"`
	Tag              string  `json:"tag" validate:"max=255"`
	IsPublished      bool    `json:"isPublished"`
}

// UpdateRequest is an alias — same fields as CreateRequest.
type UpdateRequest = CreateRequest

// Filter holds query parameters for listing events (used by repository & service).
type Filter struct {
	Search        string
	Divisions     []string
	Years         []string
	Statuses      []string
	PublishedOnly bool
	SortBy        string
	SortOrder     string
	Limit         int
	Offset        int
}

// EventResponse is the enriched payload returned by the public detail endpoint.
type EventResponse struct {
	EventID          int64      `json:"eventID"`
	EventTitle       string     `json:"eventTitle"`
	EventSlug        string     `json:"eventSlug"`
	EventDivision    string     `json:"eventDivision"`
	EventContent     string     `json:"eventContent"`
	EventImage       *string    `json:"eventImage"`
	StartDate        *time.Time `json:"startDate"`
	EndDate          *time.Time `json:"endDate"`
	CloseRegistDate  *time.Time `json:"closeRegistDate"`
	Location         *string    `json:"location"`
	Place            *string    `json:"place"`
	LocationLink     *string    `json:"locationLink"`
	RegistrationLink *string    `json:"registrationLink"`
	DocumentLink     *string    `json:"documentLink"`
	PresentationLink *string    `json:"presentationLink"`
	ContactPerson1   *string    `json:"contactPerson1"`
	NameCp1          *string    `json:"nameCp1"`
	ContactPerson2   *string    `json:"contactPerson2"`
	NameCp2          *string    `json:"nameCp2"`
	Tag              *string    `json:"tag"`
	IsPublished      bool       `json:"isPublished"`
	ViewCount        int64      `json:"viewCount"`
	Status           string     `json:"status"`
	RegistOpen       bool       `json:"registOpen"`
	AuthorID         int64      `json:"authorID"`
}
