// Package comment_dto memuat DTO request/response modul comment. Murni struct data.
package comment_dto

// CreateRequest is the body for creating a new comment or reply.
type CreateRequest struct {
	ContentType string `json:"contentType" validate:"required,oneof=article news event catalogBook"`
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

// CMSFilter menampung parameter filter khusus endpoint CMS list (di luar
// dto.ListQuery yang sudah menampung search/page/limit/sort) — dipisah dari
// CMSListFilter (dipakai repository) supaya signature service.CMSList tidak
// terus bertambah parameter positional setiap kali kolom filter baru
// ditambahkan. ContentTypes multi-select (checkbox) dikirim frontend sebagai
// query param comma-separated, di-parse handler lewat dto.ParseCSV. Author
// dipisah dari dto.ListQuery.Search (yang cuma menyaring commentText) —
// keduanya kolom pencarian TERPISAH di CMS (Komentar vs Penulis), bukan
// gabungan OR seperti sebelumnya.
type CMSFilter struct {
	ContentTypes []string
	Author       string
	DateFrom     string
	DateTo       string
}

// CMSListFilter narrows the admin comment list (top-level comments across
// all content types, optionally filtered to one or more contentTypes, an
// author name, and/or a createdDate range).
type CMSListFilter struct {
	ContentTypes []string
	Search       string // LIKE terhadap commentText — filter kolom "Komentar" CMS
	Author       string // LIKE terhadap nama penulis — filter kolom "Penulis" CMS
	DateFrom     string // "YYYY-MM-DD", inklusif
	DateTo       string // "YYYY-MM-DD", inklusif
	Limit        int
	Offset       int
	OrderBy      string
}

// Response is the recursive comment shape sent to the frontend — the public
// thread endpoint and the admin detail endpoint both reuse it. AuthorEmail
// and ContentTitle are ONLY ever populated by comment_service.CMSGet (gated
// behind comment.view) — never by PublicList/Create/Update, since author
// email must not leak to the public comment thread.
type Response struct {
	CommentID    int64        `json:"commentID"`
	ContentType  string       `json:"contentType"`
	ContentID    int64        `json:"contentID"`
	ContentTitle string       `json:"contentTitle,omitempty"`
	CommentText  string       `json:"commentText"`
	MediaURL     string       `json:"mediaURL,omitempty"`
	MediaType    string       `json:"mediaType,omitempty"`
	ParentID     *int64       `json:"parentID"`
	IsOwner      bool         `json:"isOwner"`
	CreatedDate  string       `json:"createdDate"`
	Author       AuthorDTO    `json:"author"`
	AuthorEmail  string       `json:"authorEmail,omitempty"`
	Reactions    ReactionsDTO `json:"reactions"`
	Mentions     []AuthorDTO  `json:"mentions"`
	Replies      []Response   `json:"replies"`
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
