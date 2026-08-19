// Package event_model holds the Event entity for the event module.
package event_model

import "time"

// EventStatus represents the computed lifecycle state of an event.
type EventStatus string

const (
	StatusUpcoming EventStatus = "upcoming"
	StatusOngoing  EventStatus = "ongoing"
	StatusPast     EventStatus = "past"
)

// Event represents a single row in ms_event.
type Event struct {
	EventID          int64      `gorm:"column:eventID;primaryKey;autoIncrement" json:"eventID"`
	EventTitle       string     `gorm:"column:eventTitle" json:"eventTitle"`
	EventSlug        string     `gorm:"column:eventSlug;uniqueIndex" json:"eventSlug"`
	EventDivision    string     `gorm:"column:eventDivision" json:"eventDivision"`
	EventContent     string     `gorm:"column:eventContent" json:"eventContent"`
	EventImage       *string    `gorm:"column:eventImage" json:"eventImage"`
	StartDate        *time.Time `gorm:"column:startDate" json:"startDate"`
	EndDate          *time.Time `gorm:"column:endDate" json:"endDate"`
	CloseRegistDate  *time.Time `gorm:"column:closeRegistDate" json:"closeRegistDate"`
	Location         *string    `gorm:"column:location" json:"location"`
	Place            *string    `gorm:"column:place" json:"place"`
	LocationLink     *string    `gorm:"column:locationLink" json:"locationLink"`
	RegistrationLink *string    `gorm:"column:registrationLink" json:"registrationLink"`
	DocumentLink     *string    `gorm:"column:documentLink" json:"documentLink"`
	PresentationLink *string    `gorm:"column:presentationLink" json:"presentationLink"`
	ContactPerson1   *string    `gorm:"column:contactPerson1" json:"contactPerson1"`
	NameCp1          *string    `gorm:"column:nameCp1" json:"nameCp1"`
	ContactPerson2   *string    `gorm:"column:contactPerson2" json:"contactPerson2"`
	NameCp2          *string    `gorm:"column:nameCp2" json:"nameCp2"`
	Tag              *string    `gorm:"column:tag" json:"tag"`
	IsPublished      bool       `gorm:"column:isPublished;default:false" json:"isPublished"`
	ViewCount        int64      `gorm:"column:viewCount;default:0" json:"viewCount"`
	AuthorID         int64      `gorm:"column:authorID" json:"authorID"`
	CreatedDate      time.Time  `gorm:"column:createdDate;autoCreateTime" json:"createdDate"`
	CreatedBy        *int64     `gorm:"column:createdBy" json:"createdBy,omitempty"`
	UpdatedDate      *time.Time `gorm:"column:updatedDate;autoUpdateTime" json:"updatedDate,omitempty"`
	UpdatedBy        *int64     `gorm:"column:updatedBy" json:"updatedBy,omitempty"`
}

func (Event) TableName() string { return "ms_event" }
