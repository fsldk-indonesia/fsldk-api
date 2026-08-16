package comment_repository

import (
	"context"
	"errors"
	"time"

	"fsldk-api/modules/comment/comment_dto"
	"fsldk-api/modules/comment/comment_model"

	"gorm.io/gorm"
)

const selectCols = "cm.commentID, cm.contentType, cm.contentID, cm.parentID, cm.commentText, cm.mediaURL, cm.mediaType, " +
	"cm.createdDate, cm.createdBy, u.fullName AS authorName, u.photoURL AS authorPhoto, cm.updatedDate, cm.updatedBy"

// RepositoryImpl adalah implementasi Repository berbasis GORM.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) baseQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Table("ms_comment cm").
		Joins("JOIN ms_user u ON u.userID = cm.createdBy")
}

func (r *RepositoryImpl) ListByContent(ctx context.Context, contentType string, contentID int64) ([]comment_model.Comment, error) {
	var out []comment_model.Comment
	err := r.baseQuery(ctx).Select(selectCols).
		Where("cm.contentType = ? AND cm.contentID = ?", contentType, contentID).
		Order("cm.createdDate ASC").
		Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) FindByID(ctx context.Context, id int64) (comment_model.Comment, error) {
	var c comment_model.Comment
	err := r.baseQuery(ctx).Select(selectCols).Where("cm.commentID = ?", id).Take(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return comment_model.Comment{}, ErrNotFound
	}
	return c, err
}

func (r *RepositoryImpl) FindByIDs(ctx context.Context, ids []int64) ([]comment_model.Comment, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var out []comment_model.Comment
	err := r.baseQuery(ctx).Select(selectCols).
		Where("cm.commentID IN ?", ids).
		Order("cm.createdDate ASC").
		Find(&out).Error
	return out, err
}

// DepthOf returns the depth of the given comment: 0 for a top-level comment,
// 1 for a reply (replies-of-replies are no longer allowed — see
// comment_service.Create). Walks parentID one hop at a time — business rule
// (comment_service) caps real depth at 1, so a small bounded loop is simpler
// and more portable than a recursive CTE.
func (r *RepositoryImpl) DepthOf(ctx context.Context, id int64) (int, error) {
	type row struct {
		ParentID *int64 `gorm:"column:parentID"`
	}
	depth := 0
	currentID := id
	for i := 0; i < 10; i++ { // hard safety cap; real depth is capped at 1 by business rule
		var rr row
		err := r.db.WithContext(ctx).Table("ms_comment").
			Select("parentID").Where("commentID = ?", currentID).Take(&rr).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if i == 0 {
				return 0, ErrNotFound
			}
			return depth, nil
		}
		if err != nil {
			return 0, err
		}
		if rr.ParentID == nil {
			return depth, nil
		}
		depth++
		currentID = *rr.ParentID
	}
	return depth, nil
}

// CollectDescendantIDs returns id plus every reply/reply-of-reply beneath it
// (breadth-first), regardless of how deep the comment itself already sits.
func (r *RepositoryImpl) CollectDescendantIDs(ctx context.Context, id int64) ([]int64, error) {
	ids := []int64{id}
	frontier := []int64{id}
	for len(frontier) > 0 {
		var children []int64
		if err := r.db.WithContext(ctx).Table("ms_comment").
			Where("parentID IN ?", frontier).Pluck("commentID", &children).Error; err != nil {
			return nil, err
		}
		if len(children) == 0 {
			break
		}
		ids = append(ids, children...)
		frontier = children
	}
	return ids, nil
}

func (r *RepositoryImpl) IDsByContent(ctx context.Context, contentType string, contentID int64) ([]int64, error) {
	var ids []int64
	err := r.db.WithContext(ctx).Table("ms_comment").
		Where("contentType = ? AND contentID = ?", contentType, contentID).
		Pluck("commentID", &ids).Error
	return ids, err
}

func (r *RepositoryImpl) MediaPathsByIDs(ctx context.Context, ids []int64) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var urls []string
	err := r.db.WithContext(ctx).Table("ms_comment").
		Where("commentID IN ? AND mediaType = ? AND mediaURL IS NOT NULL", ids, "image").
		Pluck("mediaURL", &urls).Error
	return urls, err
}

func (r *RepositoryImpl) CMSList(ctx context.Context, f comment_dto.CMSListFilter) ([]comment_model.Comment, int64, error) {
	q := r.baseQuery(ctx).Where("cm.parentID IS NULL")
	if f.ContentType != "" {
		q = q.Where("cm.contentType = ?", f.ContentType)
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("(cm.commentText LIKE ? OR u.fullName LIKE ?)", like, like)
	}

	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []comment_model.Comment
	err := q.Select(selectCols).Order(f.OrderBy).Limit(f.Limit).Offset(f.Offset).Find(&out).Error
	return out, total, err
}

func (r *RepositoryImpl) Create(ctx context.Context, c comment_model.Comment) (int64, error) {
	values := map[string]interface{}{
		"contentType": c.ContentType,
		"contentID":   c.ContentID,
		"parentID":    c.ParentID,
		"commentText": c.CommentText,
		"mediaURL":    c.MediaURL,
		"mediaType":   c.MediaType,
		"createdDate": time.Now(),
		"createdBy":   c.CreatedBy,
	}
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("ms_comment").Create(values).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) Update(ctx context.Context, id int64, commentText, mediaURL, mediaType *string, updatedBy int64) error {
	return r.db.WithContext(ctx).Table("ms_comment").Where("commentID = ?", id).Updates(map[string]interface{}{
		"commentText": commentText,
		"mediaURL":    mediaURL,
		"mediaType":   mediaType,
		"updatedDate": time.Now(),
		"updatedBy":   updatedBy,
	}).Error
}

// Delete removes a single comment row. FK ON DELETE CASCADE on parentID and
// on tr_comment_reaction.commentID takes care of replies and reactions —
// callers only need to remove the target row itself.
func (r *RepositoryImpl) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Exec("DELETE FROM ms_comment WHERE commentID = ?", id).Error
}

// DeleteByContent removes every comment (top-level and replies alike, since
// every row carries contentType/contentID) attached to one piece of content.
// Called by article/news/event services after they delete the content
// itself — there is no FK from ms_comment to ms_article/ms_news/ms_event to
// cascade this automatically (see techspec §3.1a).
func (r *RepositoryImpl) DeleteByContent(ctx context.Context, contentType string, contentID int64) error {
	return r.db.WithContext(ctx).Exec(
		"DELETE FROM ms_comment WHERE contentType = ? AND contentID = ?", contentType, contentID).Error
}

func (r *RepositoryImpl) ReactionCounts(ctx context.Context, commentIDs []int64) (map[int64]map[string]int64, error) {
	out := make(map[int64]map[string]int64)
	if len(commentIDs) == 0 {
		return out, nil
	}
	type row struct {
		CommentID    int64
		ReactionType string
		Cnt          int64
	}
	var rows []row
	err := r.db.WithContext(ctx).Table("tr_comment_reaction").
		Select("commentID, reactionType, COUNT(*) AS cnt").
		Where("commentID IN ?", commentIDs).
		Group("commentID, reactionType").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, rr := range rows {
		if out[rr.CommentID] == nil {
			out[rr.CommentID] = make(map[string]int64)
		}
		out[rr.CommentID][rr.ReactionType] = rr.Cnt
	}
	return out, nil
}

func (r *RepositoryImpl) UserReactionTypes(ctx context.Context, commentIDs []int64, userID int64) (map[int64][]string, error) {
	out := make(map[int64][]string)
	if len(commentIDs) == 0 || userID == 0 {
		return out, nil
	}
	type row struct {
		CommentID    int64
		ReactionType string
	}
	var rows []row
	err := r.db.WithContext(ctx).Table("tr_comment_reaction").
		Select("commentID, reactionType").
		Where("commentID IN ? AND userID = ?", commentIDs, userID).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, rr := range rows {
		out[rr.CommentID] = append(out[rr.CommentID], rr.ReactionType)
	}
	return out, nil
}

func (r *RepositoryImpl) ReactionExists(ctx context.Context, commentID, userID int64, reactionType string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("tr_comment_reaction").
		Where("commentID = ? AND userID = ? AND reactionType = ?", commentID, userID, reactionType).
		Count(&count).Error
	return count > 0, err
}

func (r *RepositoryImpl) CreateReaction(ctx context.Context, commentID, userID int64, reactionType string) error {
	return r.db.WithContext(ctx).Table("tr_comment_reaction").Create(map[string]interface{}{
		"commentID":    commentID,
		"userID":       userID,
		"reactionType": reactionType,
		"createdDate":  time.Now(),
	}).Error
}

func (r *RepositoryImpl) DeleteReaction(ctx context.Context, commentID, userID int64, reactionType string) error {
	return r.db.WithContext(ctx).Exec(
		"DELETE FROM tr_comment_reaction WHERE commentID = ? AND userID = ? AND reactionType = ?",
		commentID, userID, reactionType).Error
}

// SetMentions replaces the full mention list for a comment (delete-then-insert
// — simpler and safer than diffing, and mention lists are always small).
func (r *RepositoryImpl) SetMentions(ctx context.Context, commentID int64, userIDs []int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM tr_comment_mention WHERE commentID = ?", commentID).Error; err != nil {
			return err
		}
		seen := make(map[int64]bool, len(userIDs))
		for _, uid := range userIDs {
			if uid <= 0 || seen[uid] {
				continue
			}
			seen[uid] = true
			if err := tx.Table("tr_comment_mention").Create(map[string]interface{}{
				"commentID": commentID, "userID": uid, "createdDate": time.Now(),
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *RepositoryImpl) MentionsByCommentIDs(ctx context.Context, commentIDs []int64) (map[int64][]comment_model.MentionAuthor, error) {
	out := make(map[int64][]comment_model.MentionAuthor)
	if len(commentIDs) == 0 {
		return out, nil
	}
	var rows []comment_model.MentionAuthor
	err := r.db.WithContext(ctx).Table("tr_comment_mention m").
		Select("m.commentID, m.userID, u.fullName, u.photoURL").
		Joins("JOIN ms_user u ON u.userID = m.userID").
		Where("m.commentID IN ?", commentIDs).
		Order("m.mentionID ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.CommentID] = append(out[row.CommentID], row)
	}
	return out, nil
}
