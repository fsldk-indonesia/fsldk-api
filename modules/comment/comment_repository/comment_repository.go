// Package comment_repository adalah lapisan akses data modul comment (GORM).
package comment_repository

import (
	"context"
	"errors"

	"fsldk-api/modules/comment/comment_dto"
	"fsldk-api/modules/comment/comment_model"
)

// ErrNotFound dikembalikan bila komentar tidak ditemukan.
var ErrNotFound = errors.New("komentar tidak ditemukan")

// Repository adalah kontrak akses data komentar.
type Repository interface {
	// Thread
	ListByContent(ctx context.Context, contentType string, contentID int64) ([]comment_model.Comment, error)
	FindByID(ctx context.Context, id int64) (comment_model.Comment, error)
	FindByIDs(ctx context.Context, ids []int64) ([]comment_model.Comment, error)
	DepthOf(ctx context.Context, id int64) (int, error)
	CollectDescendantIDs(ctx context.Context, id int64) ([]int64, error)
	IDsByContent(ctx context.Context, contentType string, contentID int64) ([]int64, error)
	MediaPathsByIDs(ctx context.Context, ids []int64) ([]string, error)

	// Admin
	CMSList(ctx context.Context, f comment_dto.CMSListFilter) ([]comment_model.Comment, int64, error)

	// Write
	Create(ctx context.Context, c comment_model.Comment) (int64, error)
	Update(ctx context.Context, id int64, commentText, mediaURL, mediaType *string, updatedBy int64) error
	Delete(ctx context.Context, id int64) error
	DeleteByContent(ctx context.Context, contentType string, contentID int64) error

	// Reactions
	ReactionCounts(ctx context.Context, commentIDs []int64) (map[int64]map[string]int64, error)
	UserReactionTypes(ctx context.Context, commentIDs []int64, userID int64) (map[int64][]string, error)
	ReactionExists(ctx context.Context, commentID, userID int64, reactionType string) (bool, error)
	CreateReaction(ctx context.Context, commentID, userID int64, reactionType string) error
	DeleteReaction(ctx context.Context, commentID, userID int64, reactionType string) error

	// Mentions
	SetMentions(ctx context.Context, commentID int64, userIDs []int64) error
	MentionsByCommentIDs(ctx context.Context, commentIDs []int64) (map[int64][]comment_model.MentionAuthor, error)
}
