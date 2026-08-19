// Package comment_dto memuat DTO request/response modul comment. Murni struct data.
package comment_dto

// CreateRequest is the body for creating a new comment or reply.
type CreateRequest struct {
	ContentType string `json:"contentType" validate:"required,oneof=article news event"`
	ContentID   int64  `json:"contentID" validate:"required,min=1"`
	ParentID    *int64 `json:"parentID"`
	CommentText string `json:"commentText" validate:"max=2000"`
	MediaURL    string `json:"mediaURL" validate:"max=500"`
	MediaType   string `json:"mediaType" validate:"omitempty,oneof=image gif sticker"`
	// MentionedUserIDs are the users the composer picked from the @mention
	// autocomplete — deliberately structured (not parsed back out of
	// commentText) so the pill rendering is always accurate. A user may
	// mention themselves.
	MentionedUserIDs []int64 `json:"mentionedUserIDs" validate:"omitempty,max=20,dive,gt=0"`
}

// UpdateRequest is the body for editing an existing comment.
type UpdateRequest struct {
	CommentText      string  `json:"commentText" validate:"max=2000"`
	MediaURL         string  `json:"mediaURL" validate:"max=500"`
	MediaType        string  `json:"mediaType" validate:"omitempty,oneof=image gif sticker"`
	MentionedUserIDs []int64 `json:"mentionedUserIDs" validate:"omitempty,max=20,dive,gt=0"`
}

// ReactRequest is the body for toggling a reaction on a comment.
type ReactRequest struct {
	ReactionType string `json:"reactionType" validate:"required,oneof=like dislike love heart_eyes laughing rage slight_smile"`
}

// BulkDeleteRequest is the body for deleting multiple comments at once (admin).
type BulkDeleteRequest struct {
	IDs []int64 `json:"ids" validate:"required,min=1"`
}

// CMSListFilter narrows the admin comment list (top-level comments across
// all content types, optionally filtered to one contentType).
type CMSListFilter struct {
	ContentType string
	Search      string
	Limit       int
	Offset      int
	OrderBy     string
}

// Response is the recursive comment shape sent to the frontend — the public
// thread endpoint and the admin detail endpoint both reuse it.
type Response struct {
	CommentID   int64        `json:"commentID"`
	ContentType string       `json:"contentType"`
	ContentID   int64        `json:"contentID"`
	CommentText string       `json:"commentText"`
	MediaURL    string       `json:"mediaURL,omitempty"`
	MediaType   string       `json:"mediaType,omitempty"`
	ParentID    *int64       `json:"parentID"`
	IsOwner     bool         `json:"isOwner"`
	CreatedDate string       `json:"createdDate"`
	Author      AuthorDTO    `json:"author"`
	Reactions   ReactionsDTO `json:"reactions"`
	Mentions    []AuthorDTO  `json:"mentions"`
	Replies     []Response   `json:"replies"`
}

// AuthorDTO is the comment author summary embedded in Response.
type AuthorDTO struct {
	UserID int64  `json:"userID"`
	Name   string `json:"name"`
	Photo  string `json:"photo,omitempty"`
}

// ReactionsDTO summarizes per-type reaction counts and the caller's own active types.
type ReactionsDTO struct {
	Counts    map[string]int64 `json:"counts"`
	UserTypes []string         `json:"userTypes"`
}

// GifItem is one GIPHY search/trending result.
type GifItem struct {
	ID      string `json:"id"`
	Preview string `json:"preview"`
	URL     string `json:"url"`
	Title   string `json:"title"`
}

// GifCategory is one GIPHY trending category.
type GifCategory struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}
