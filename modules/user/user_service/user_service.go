// Package user_service memuat logika bisnis modul user.
package user_service

import (
	"context"

	"fsldk-api/base/dto"
	"fsldk-api/modules/user/user_dto"
)

// Service adalah kontrak logika bisnis pengguna.
type Service interface {
	List(ctx context.Context, q dto.ListQuery, roleID int64) ([]user_dto.Response, int, error)
	// SearchMentionable returns a minimal active-user summary for the
	// @mention autocomplete — any verified user can call this, unlike List
	// which requires user.view.
	SearchMentionable(ctx context.Context, search string, limit int) ([]user_dto.MentionSearchResult, error)
	Get(ctx context.Context, id int64) (user_dto.Response, error)
	Create(ctx context.Context, req user_dto.CreateRequest, actorID int64) (user_dto.Response, error)
	Update(ctx context.Context, id int64, req user_dto.UpdateRequest, actorID int64) (user_dto.Response, error)
	SetStatus(ctx context.Context, id int64, active bool, actorID int64) error
	Delete(ctx context.Context, id, actorID int64) error
}
