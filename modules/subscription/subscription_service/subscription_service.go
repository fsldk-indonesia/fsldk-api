// Package subscription_service provides business logic for newsletter subscriptions.
package subscription_service

import (
	"context"

	"fsldk-api/base/dto"
	"fsldk-api/modules/subscription/subscription_dto"
)

// Service defines business operations for newsletter subscribers.
type Service interface {
	// Subscribe handles a public subscribe request (new or re-subscribe).
	// isResubscribe reports whether an inactive subscriber was reactivated.
	Subscribe(ctx context.Context, email string) (isResubscribe bool, err error)
	// Unsubscribe deactivates a subscriber from the unsubscribe link embedded
	// in the welcome email — token must match the stored per-subscriber value.
	Unsubscribe(ctx context.Context, email, token string) error
	BulkAdd(ctx context.Context, rawEmails string) (*subscription_dto.BulkAddResult, error)
	List(ctx context.Context, q dto.ListQuery, isActive *bool, from, to string) ([]subscription_dto.SubscriberResponse, int, error)
	GetByID(ctx context.Context, id int64) (*subscription_dto.SubscriberResponse, error)
	Update(ctx context.Context, id int64, req subscription_dto.UpdateSubscriberRequest) (*subscription_dto.SubscriberResponse, error)
	Delete(ctx context.Context, id int64) error
	BulkDelete(ctx context.Context, ids []int64) error
}
