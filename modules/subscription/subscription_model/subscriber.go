// Package subscription_model defines the database entity for newsletter subscribers.
package subscription_model

import "time"

// Subscriber represents a newsletter subscriber (public footer/contact form,
// managed from the CMS Subscription menu).
type Subscriber struct {
	SubscriberID     int64      `gorm:"column:subscriberID;primaryKey;autoIncrement"`
	Email            string     `gorm:"column:email;not null"`
	IsActive         bool       `gorm:"column:isActive;default:true"`
	UnsubscribeToken string     `gorm:"column:unsubscribeToken"`
	SubscribedDate   time.Time  `gorm:"column:subscribedDate"`
	UnsubscribedDate *time.Time `gorm:"column:unsubscribedDate"`
	CreatedDate      time.Time  `gorm:"column:createdDate;autoCreateTime"`
}

// TableName returns the database table name.
func (Subscriber) TableName() string {
	return "tr_subscriber"
}
