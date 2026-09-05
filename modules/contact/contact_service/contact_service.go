// Package contact_service provides business logic for contact inquiries.
package contact_service

import (
	"context"

	"fsldk-api/modules/contact/contact_dto"
)

// Service defines business operations for contact inquiries.
type Service interface {
	Send(ctx context.Context, req contact_dto.SendContactRequest, ip string) error
	GetByID(ctx context.Context, id int64) (*contact_dto.ContactDetail, error)
	List(ctx context.Context, q contact_dto.ContactListQuery) (*contact_dto.ContactListResponse, error)
	MarkRead(ctx context.Context, id int64) error
	Delete(ctx context.Context, id int64) error
	CountUnread(ctx context.Context) (int, error)
}
