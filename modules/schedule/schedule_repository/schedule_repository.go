// Package schedule_repository is the data access layer for the schedule module (GORM).
package schedule_repository

import (
	"context"
	"errors"

	"fsldk-api/modules/schedule/schedule_dto"
	"fsldk-api/modules/schedule/schedule_model"
)

// ErrNotFound is returned when a schedule cannot be found.
var ErrNotFound = errors.New("schedule not found")

// Repository is the data access contract for schedules.
type Repository interface {
	List(ctx context.Context, f schedule_dto.Filter) ([]schedule_model.Schedule, int64, error)
	FindByID(ctx context.Context, id int64) (schedule_model.Schedule, error)
	Create(ctx context.Context, s schedule_model.Schedule, actorID int64) (int64, error)
	Update(ctx context.Context, id int64, s schedule_model.Schedule, actorID int64) error
	SetActive(ctx context.Context, id int64, isActive bool, actorID int64) error
	Delete(ctx context.Context, id int64) error
}
