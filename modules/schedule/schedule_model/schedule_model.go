// Package schedule_model holds schedule module entities. Pure data, no methods.
package schedule_model

import "time"

// Schedule represents one ms_schedule row.
//
// startDate/endDate are DATE columns scanned into time.Time (the DSN sets
// parseTime=true). startTime/endTime are TIME columns, which the MySQL driver
// returns as strings ("15:04:05") — the service trims them to "15:04" for the
// response. endDate is nil for a single-day activity.
type Schedule struct {
	ScheduleID    int64      `gorm:"column:scheduleID;primaryKey" json:"scheduleID"`
	Title         string     `gorm:"column:title" json:"title"`
	Description   *string    `gorm:"column:description" json:"description"`
	Category      string     `gorm:"column:category" json:"category"`
	StartDate     time.Time  `gorm:"column:startDate" json:"startDate"`
	EndDate       *time.Time `gorm:"column:endDate" json:"endDate"`
	IsAllDay      bool       `gorm:"column:isAllDay" json:"isAllDay"`
	StartTime     *string    `gorm:"column:startTime" json:"startTime"`
	EndTime       *string    `gorm:"column:endTime" json:"endTime"`
	Location      *string    `gorm:"column:location" json:"location"`
	Organizer     *string    `gorm:"column:organizer" json:"organizer"`
	ContactPerson *string    `gorm:"column:contactPerson" json:"contactPerson"`
	URL           *string    `gorm:"column:url" json:"url"`
	IsActive      bool       `gorm:"column:isActive" json:"isActive"`
	CreatedDate   time.Time  `gorm:"column:createdDate" json:"createdDate"`
	UpdatedDate   *time.Time `gorm:"column:updatedDate" json:"updatedDate"`
}
