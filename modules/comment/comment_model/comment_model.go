// Package comment_model memuat entitas modul comment. Murni struct data.
package comment_model

import "time"

// Comment merepresentasikan satu baris ms_comment (dengan kolom join penulis).
// contentType/contentID menunjuk ke konten manapun (article, news, ...) tanpa
// foreign key — lihat techspec §3.1a untuk trade-off desain ini.
type Comment struct {
	CommentID   int64      `gorm:"column:commentID;primaryKey" json:"commentID"`
	ContentType string     `gorm:"column:contentType" json:"contentType"`
	ContentID   int64      `gorm:"column:contentID" json:"contentID"`
	ParentID    *int64     `gorm:"column:parentID" json:"parentID"`
	CommentText *string    `gorm:"column:commentText" json:"commentText"`
	MediaURL    *string    `gorm:"column:mediaURL" json:"mediaURL"`
	MediaType   *string    `gorm:"column:mediaType" json:"mediaType"`
	CreatedDate time.Time  `gorm:"column:createdDate" json:"createdDate"`
	CreatedBy   int64      `gorm:"column:createdBy" json:"createdBy"`
	AuthorName  string     `gorm:"column:authorName;->" json:"authorName"`
	AuthorPhoto *string    `gorm:"column:authorPhoto;->" json:"authorPhoto"`
	UpdatedDate *time.Time `gorm:"column:updatedDate" json:"updatedDate"`
	UpdatedBy   *int64     `gorm:"column:updatedBy" json:"updatedBy"`
}

// MentionAuthor adalah hasil join tr_comment_mention + ms_user — ringkasan
// pengguna yang di-@mention pada satu komentar, dipakai membangun
// comment_dto.Response.Mentions.
type MentionAuthor struct {
	CommentID int64   `gorm:"column:commentID"`
	UserID    int64   `gorm:"column:userID"`
	FullName  string  `gorm:"column:fullName"`
	PhotoURL  *string `gorm:"column:photoURL"`
}

// CommentReaction merepresentasikan satu baris tr_comment_reaction.
type CommentReaction struct {
	ReactionID   int64     `gorm:"column:reactionID;primaryKey" json:"reactionID"`
	CommentID    int64     `gorm:"column:commentID" json:"commentID"`
	UserID       int64     `gorm:"column:userID" json:"userID"`
	ReactionType string    `gorm:"column:reactionType" json:"reactionType"`
	CreatedDate  time.Time `gorm:"column:createdDate" json:"createdDate"`
}

// ReactionTypes is the fixed set of allowed reaction types — the single
// source of truth for validation (comment_dto tags). The frontend keeps a
// manually-synced copy in comment.constants.ts; update both together.
var ReactionTypes = []string{"like", "dislike", "love", "heart_eyes", "laughing", "rage", "slight_smile"}

// ValidContentTypes is the whitelist of content types comments can attach
// to. Add an entry here (and update the matching `oneof=...` validate tags
// in comment_dto) whenever a new content module needs comments — no
// migration required since contentType is a plain VARCHAR column.
var ValidContentTypes = []string{"article", "news", "event"}
