// Package contact_model defines the database entity for contact messages.
package contact_model

import "time"

// ContactMessage represents a public contact inquiry submitted to FSLDK.
type ContactMessage struct {
	MessageID   int64     `gorm:"column:messageID;primaryKey;autoIncrement"`
	SenderName  string    `gorm:"column:senderName;not null"`
	Email       string    `gorm:"column:email;not null"`
	Subject     string    `gorm:"column:subject;not null"`
	Message     string    `gorm:"column:message;not null"`
	IPAddress   *string   `gorm:"column:ipAddress"`
	IsRead      bool      `gorm:"column:isRead;default:false"`
	CreatedDate time.Time `gorm:"column:createdDate;autoCreateTime"`
}

// TableName returns the database table name.
func (ContactMessage) TableName() string {
	return "tr_contact_message"
}
