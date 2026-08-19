// Package comment_service memuat logika bisnis modul comment.
package comment_service

import (
	"context"

	"fsldk-api/base/dto"
	"fsldk-api/modules/comment/comment_dto"
)

// Service adalah kontrak logika bisnis komentar.
type Service interface {
	// Public thread + self-service actions (any verified logged-in user).
	PublicList(ctx context.Context, contentType string, contentID, currentUserID int64) ([]comment_dto.Response, error)
	Create(ctx context.Context, req comment_dto.CreateRequest, userID int64) (comment_dto.Response, error)
	Update(ctx context.Context, id int64, req comment_dto.UpdateRequest, userID int64, isModerator bool) (comment_dto.Response, error)
	Delete(ctx context.Context, id, userID int64, isModerator bool) error
	React(ctx context.Context, commentID, userID int64, reactionType string) (comment_dto.ReactionsDTO, error)
	GifSearch(ctx context.Context, query, tab string) ([]comment_dto.GifItem, error)
	GifCategories(ctx context.Context) ([]comment_dto.GifCategory, error)

	// Admin moderation (comment.view / comment.delete).
	CMSList(ctx context.Context, q dto.ListQuery, contentType string) ([]comment_dto.Response, int, error)
	CMSGet(ctx context.Context, id, currentUserID int64) (comment_dto.Response, error)
	BulkDelete(ctx context.Context, ids []int64) error

	// DeleteByContent removes every comment attached to one piece of content.
	// Called by article_service/news_service/event_service (via the
	// CommentCleaner interface each declares) right after they delete the
	// content itself — see techspec §8.4.
	DeleteByContent(ctx context.Context, contentType string, contentID int64) error
}
