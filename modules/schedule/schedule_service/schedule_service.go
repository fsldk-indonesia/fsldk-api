// Package schedule_service holds schedule module business logic.
package schedule_service

import (
	"context"

	"fsldk-api/base/dto"
	"fsldk-api/modules/schedule/schedule_dto"
)

// Service is the business logic contract for schedules.
type Service interface {
	// PublicList returns active schedules overlapping the [from, to] date
	// window (both "YYYY-MM-DD"). The window is capped at 92 days.
	PublicList(ctx context.Context, from, to string) ([]schedule_dto.Response, error)
	CMSList(ctx context.Context, q dto.ListQuery, f schedule_dto.Filter) ([]schedule_dto.Response, int, error)
	Get(ctx context.Context, id int64) (schedule_dto.Response, error)

	Create(ctx context.Context, req schedule_dto.Request, actorID int64) (schedule_dto.Response, error)
	Update(ctx context.Context, id int64, req schedule_dto.Request, actorID int64) (schedule_dto.Response, error)
	SetActive(ctx context.Context, id int64, isActive bool, actorID int64) error
	Delete(ctx context.Context, id int64) error
}
