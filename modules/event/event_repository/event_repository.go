// Package event_repository is the data-access layer for the event module (GORM).
package event_repository

import (
	"context"
	"errors"

	"fsldk-api/modules/event/event_dto"
	"fsldk-api/modules/event/event_model"
)

// ErrNotFound is returned when an event cannot be found.
var ErrNotFound = errors.New("event not found")

// Repository defines the data-access contract for events.
type Repository interface {
	List(ctx context.Context, f event_dto.Filter) ([]event_model.Event, int64, error)
	FindByID(ctx context.Context, id int64) (event_model.Event, error)
	FindBySlug(ctx context.Context, slug string) (event_model.Event, error)
	SlugExists(ctx context.Context, slug string, exceptID int64) (bool, error)
	Create(ctx context.Context, e event_model.Event, authorID int64) (int64, error)
	Update(ctx context.Context, id int64, e event_model.Event, updatedBy int64) error
	Delete(ctx context.Context, id int64) error
	IncrementViewCount(ctx context.Context, id int64) error
}
