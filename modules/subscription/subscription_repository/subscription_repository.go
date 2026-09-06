// Package subscription_repository provides data access operations for newsletter subscribers.
package subscription_repository

import (
	"context"
	"errors"

	"fsldk-api/base/dto"
	"fsldk-api/modules/subscription/subscription_model"
)

// ErrNotFound is returned when a subscriber record is not found.
var ErrNotFound = errors.New("subscriber not found")

// Repository defines data access methods for newsletter subscribers.
type Repository interface {
	FindByEmail(ctx context.Context, email string) (*subscription_model.Subscriber, error)
	FindByID(ctx context.Context, id int64) (*subscription_model.Subscriber, error)
	// FindAll lists subscribers with the standard search/sort/pagination
	// envelope plus isActive/subscribedDate-range filters specific to this
	// module. from/to are "YYYY-MM-DD"; empty strings mean unbounded.
	FindAll(ctx context.Context, q dto.ListQuery, isActive *bool, from, to string) ([]subscription_model.Subscriber, int, error)
	Create(ctx context.Context, sub *subscription_model.Subscriber) error
	Update(ctx context.Context, sub *subscription_model.Subscriber) error
	EmailExistsExcluding(ctx context.Context, email string, excludeID int64) (bool, error)
	Delete(ctx context.Context, id int64) error
	DeleteMany(ctx context.Context, ids []int64) error
}
