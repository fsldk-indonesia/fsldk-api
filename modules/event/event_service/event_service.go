// Package event_service contains the business logic for the event module.
package event_service

import (
	"context"

	"fsldk-api/base/dto"
	"fsldk-api/modules/event/event_dto"
	"fsldk-api/modules/event/event_model"
)

// Service defines the business-logic contract for events.
type Service interface {
	// PublicList returns published events for the landing page.
	PublicList(ctx context.Context, q dto.ListQuery, divisions, years, statuses []string, sort string) ([]event_dto.EventResponse, int, error)
	// PublicDetail returns a published event by slug and increments its view count.
	PublicDetail(ctx context.Context, slug string) (event_dto.EventResponse, error)
	// CMSList returns all events (published or not) for the CMS dashboard.
	CMSList(ctx context.Context, q dto.ListQuery, division string) ([]event_model.Event, int, error)
	// CMSGet returns any event by ID for CMS editing.
	CMSGet(ctx context.Context, id int64) (event_model.Event, error)
	// Create validates, slugifies, and persists a new event.
	Create(ctx context.Context, req event_dto.CreateRequest, authorID int64) (event_model.Event, error)
	// Update validates and persists changes to an existing event.
	Update(ctx context.Context, id int64, req event_dto.UpdateRequest, updatedBy int64) (event_model.Event, error)
	// Delete removes an event by ID.
	Delete(ctx context.Context, id int64) error
}
