// Package contact_dto contains data transfer objects for contact message operations.
package contact_dto

import "time"

// SendContactRequest defines payload submitted from public contact form.
type SendContactRequest struct {
	SenderName string `json:"senderName" validate:"required,min=3,max=100"`
	Email      string `json:"email"       validate:"required,email,max=255"`
	Subject    string `json:"subject"     validate:"required,min=5,max=200"`
	Message    string `json:"message"     validate:"required,min=10,max=1000"`
}

// ContactListItem represents an item in the CMS inbox list.
type ContactListItem struct {
	MessageID   int64     `json:"messageID"`
	SenderName  string    `json:"senderName"`
	Email       string    `json:"email"`
	Subject     string    `json:"subject"`
	IsRead      bool      `json:"isRead"`
	CreatedDate time.Time `json:"createdDate"`
}

// ContactDetail represents the complete message content for viewing in CMS.
type ContactDetail struct {
	MessageID   int64     `json:"messageID"`
	SenderName  string    `json:"senderName"`
	Email       string    `json:"email"`
	Subject     string    `json:"subject"`
	Message     string    `json:"message"`
	IPAddress   *string   `json:"ipAddress,omitempty"`
	IsRead      bool      `json:"isRead"`
	CreatedDate time.Time `json:"createdDate"`
}

// ContactListQuery defines filtering and pagination parameters for the CMS inbox.
type ContactListQuery struct {
	Page      int    `form:"page"`
	Limit     int    `form:"limit"`
	Search    string `form:"search"`
	IsRead    *bool  `form:"isRead"`
	SortBy    string `form:"sort_by"`
	SortOrder string `form:"sort_order"`
}

// ContactListResponse defines the paginated list envelope.
type ContactListResponse struct {
	Data  []ContactListItem `json:"data"`
	Page  int               `json:"page"`
	Limit int               `json:"limit"`
	Total int64             `json:"total"`
}

// ReplyContactRequest defines the payload sent by CMS admin to reply to a contact inquiry via email.
type ReplyContactRequest struct {
	Subject string `json:"subject" validate:"required,min=3,max=200"`
	Message string `json:"message" validate:"required,min=5,max=5000"`
}
