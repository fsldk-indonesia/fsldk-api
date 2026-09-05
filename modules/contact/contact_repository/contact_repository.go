// Package contact_repository provides data access operations for contact messages.
package contact_repository

import (
	"context"
	"errors"

	"fsldk-api/modules/contact/contact_dto"
	"fsldk-api/modules/contact/contact_model"
)

var (
	// ErrNotFound is returned when a contact message record is not found.
	ErrNotFound = errors.New("contact message not found")
)

// Repository defines data access methods for contact messages.
type Repository interface {
	Create(ctx context.Context, msg *contact_model.ContactMessage) error
	FindByID(ctx context.Context, id int64) (*contact_model.ContactMessage, error)
	FindAll(ctx context.Context, q contact_dto.ContactListQuery) ([]contact_model.ContactMessage, int64, error)
	MarkAsRead(ctx context.Context, id int64) error
	Delete(ctx context.Context, id int64) error
	CountUnread(ctx context.Context) (int, error)
}
