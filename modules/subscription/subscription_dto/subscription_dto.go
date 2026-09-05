// Package subscription_dto contains data transfer objects for newsletter subscription operations.
package subscription_dto

import "time"

// SubscribeRequest defines the payload submitted from public forms (footer, Hubungi Kami).
type SubscribeRequest struct {
	Email string `json:"email" validate:"required,email,max=255"`
}

// BulkAddRequest defines the payload for CMS "Add Subscribers" (bulk textarea input).
type BulkAddRequest struct {
	Emails string `json:"emails" validate:"required"`
}

// BulkAddResult summarizes the outcome of a bulk add operation.
type BulkAddResult struct {
	Added   int      `json:"added"`
	Skipped []string `json:"skipped"`
	Invalid []string `json:"invalid"`
}

// UpdateSubscriberRequest defines the payload for CMS edit (email & status).
type UpdateSubscriberRequest struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	IsActive bool   `json:"isActive"`
}

// UnsubscribeRequest defines the payload for the public unsubscribe link (from email).
type UnsubscribeRequest struct {
	Email string `json:"email" validate:"required,email"`
	Token string `json:"token" validate:"required"`
}

// BulkDeleteRequest defines the payload for CMS bulk delete (Super Admin only).
type BulkDeleteRequest struct {
	IDs []int64 `json:"ids" validate:"required,min=1"`
}

// SubscriberResponse represents a subscriber row, used both in the CMS list and detail view.
type SubscriberResponse struct {
	SubscriberID     int64      `json:"subscriberID"`
	Email            string     `json:"email"`
	IsActive         bool       `json:"isActive"`
	SubscribedDate   time.Time  `json:"subscribedDate"`
	UnsubscribedDate *time.Time `json:"unsubscribedDate,omitempty"`
	CreatedDate      time.Time  `json:"createdDate"`
}
