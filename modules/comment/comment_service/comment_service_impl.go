package comment_service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/ptr"
	"fsldk-api/modules/comment/comment_dto"
	"fsldk-api/modules/comment/comment_model"
	"fsldk-api/modules/comment/comment_repository"
)

var sortColumns = map[string]string{
	"contentType": "cm.contentType",
	"createdDate": "cm.createdDate",
}

// FileDeleter is the narrow slice of pkg/upload.Uploader this service
// depends on — accepting an interface (not the concrete type) keeps
// comment_service decoupled from the upload package's implementation.
type FileDeleter interface {
	DeleteFile(publicURL string) error
}

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct {
	repo        comment_repository.Repository
	upload      FileDeleter
	giphyAPIKey string
}

// NewService membuat Service komentar.
func NewService(repo comment_repository.Repository, upload FileDeleter, giphyAPIKey string) Service {
	return &ServiceImpl{repo: repo, upload: upload, giphyAPIKey: giphyAPIKey}
}

func (s *ServiceImpl) PublicList(ctx context.Context, contentType string, contentID, currentUserID int64) ([]comment_dto.Response, error) {
	flat, err := s.repo.ListByContent(ctx, contentType, contentID)
	if err != nil {
		return nil, apperror.Internal("")
	}
	reactions, userReactions, err := s.loadReactions(ctx, commentIDs(flat), currentUserID)
	if err != nil {
		return nil, apperror.Internal("")
	}
	tree := buildTree(flat, nil, reactions, userReactions, currentUserID)
	if tree == nil {
		tree = []comment_dto.Response{}
	}
	return tree, nil
}

func (s *ServiceImpl) Create(ctx context.Context, req comment_dto.CreateRequest, userID int64) (comment_dto.Response, error) {
	if !slices.Contains(comment_model.ValidContentTypes, req.ContentType) {
		return comment_dto.Response{}, apperror.Validation("Validation Error", []apperror.FieldError{
			{Attribute: "contentType", Message: "Tipe konten tidak dikenali"},
		})
	}
	if strings.TrimSpace(req.CommentText) == "" && req.MediaURL == "" {
		return comment_dto.Response{}, apperror.Validation("Validation Error", []apperror.FieldError{
			{Attribute: "commentText", Message: "Komentar atau media wajib diisi"},
		})
	}
	if req.ParentID != nil {
		depth, err := s.repo.DepthOf(ctx, *req.ParentID)
		if err != nil {
			return comment_dto.Response{}, apperror.NotFound("Komentar induk tidak ditemukan")
		}
		if depth >= 2 {
			return comment_dto.Response{}, apperror.Validation("Validation Error", []apperror.FieldError{
				{Attribute: "parentID", Message: "Kedalaman balasan maksimal 2 level"},
			})
		}
	}

	entity := comment_model.Comment{
		ContentType: req.ContentType,
		ContentID:   req.ContentID,
		ParentID:    req.ParentID,
		CommentText: ptr.Str(req.CommentText),
		MediaURL:    ptr.Str(req.MediaURL),
		MediaType:   ptr.Str(req.MediaType),
		CreatedBy:   userID,
	}
	id, err := s.repo.Create(ctx, entity)
	if err != nil {
		return comment_dto.Response{}, apperror.Internal("Gagal menyimpan komentar")
	}
	return s.getWithThread(ctx, id, userID)
}

func (s *ServiceImpl) Update(ctx context.Context, id int64, req comment_dto.UpdateRequest, userID int64) (comment_dto.Response, error) {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return comment_dto.Response{}, apperror.NotFound("Komentar tidak ditemukan")
	}
	if existing.CreatedBy != userID {
		return comment_dto.Response{}, apperror.Forbidden("Anda tidak berhak mengubah komentar ini")
	}
	if strings.TrimSpace(req.CommentText) == "" && req.MediaURL == "" {
		return comment_dto.Response{}, apperror.Validation("Validation Error", []apperror.FieldError{
			{Attribute: "commentText", Message: "Komentar atau media wajib diisi"},
		})
	}

	oldMediaURL, oldMediaType := "", ""
	if existing.MediaURL != nil {
		oldMediaURL = *existing.MediaURL
	}
	if existing.MediaType != nil {
		oldMediaType = *existing.MediaType
	}

	if err := s.repo.Update(ctx, id, ptr.Str(req.CommentText), ptr.Str(req.MediaURL), ptr.Str(req.MediaType), userID); err != nil {
		return comment_dto.Response{}, apperror.Internal("")
	}

	// Best-effort cleanup of the previously-stored local image when it's
	// been replaced. GIF/sticker URLs are external (GIPHY), never local —
	// only mediaType "image" was ever saved via pkg/upload.
	if oldMediaType == "image" && oldMediaURL != "" && oldMediaURL != req.MediaURL {
		_ = s.upload.DeleteFile(oldMediaURL)
	}

	return s.getWithThread(ctx, id, userID)
}

func (s *ServiceImpl) Delete(ctx context.Context, id, userID int64, isModerator bool) error {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return apperror.NotFound("Komentar tidak ditemukan")
	}
	if c.CreatedBy != userID && !isModerator {
		return apperror.Forbidden("Anda tidak berhak menghapus komentar ini")
	}
	ids, err := s.repo.CollectDescendantIDs(ctx, id)
	if err != nil {
		return apperror.Internal("")
	}
	mediaPaths, _ := s.repo.MediaPathsByIDs(ctx, ids)
	// FK ON DELETE CASCADE on ms_comment.parentID and tr_comment_reaction.commentID
	// takes care of replies and reactions — deleting the root row is enough.
	if err := s.repo.Delete(ctx, id); err != nil {
		return apperror.Internal("")
	}
	for _, p := range mediaPaths {
		_ = s.upload.DeleteFile(p) // best-effort, errors are not fatal to the delete
	}
	return nil
}

func (s *ServiceImpl) React(ctx context.Context, commentID, userID int64, reactionType string) (comment_dto.ReactionsDTO, error) {
	if _, err := s.repo.FindByID(ctx, commentID); err != nil {
		return comment_dto.ReactionsDTO{}, apperror.NotFound("Komentar tidak ditemukan")
	}
	exists, err := s.repo.ReactionExists(ctx, commentID, userID, reactionType)
	if err != nil {
		return comment_dto.ReactionsDTO{}, apperror.Internal("")
	}
	if exists {
		if err := s.repo.DeleteReaction(ctx, commentID, userID, reactionType); err != nil {
			return comment_dto.ReactionsDTO{}, apperror.Internal("")
		}
	} else if err := s.repo.CreateReaction(ctx, commentID, userID, reactionType); err != nil {
		return comment_dto.ReactionsDTO{}, apperror.Internal("")
	}

	counts, userTypes, err := s.loadReactions(ctx, []int64{commentID}, userID)
	if err != nil {
		return comment_dto.ReactionsDTO{}, apperror.Internal("")
	}
	c := counts[commentID]
	if c == nil {
		c = map[string]int64{}
	}
	return comment_dto.ReactionsDTO{Counts: c, UserTypes: userTypes[commentID]}, nil
}

func (s *ServiceImpl) CMSList(ctx context.Context, q dto.ListQuery, contentType string) ([]comment_dto.Response, int, error) {
	f := comment_dto.CMSListFilter{
		ContentType: contentType,
		Search:      q.Search,
		Limit:       q.Limit,
		Offset:      q.Offset(),
		OrderBy:     q.OrderBy(sortColumns, "cm.createdDate DESC"),
	}
	flat, total, err := s.repo.CMSList(ctx, f)
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	// Top-level rows only in this list — reaction counts shown here have no
	// "isOwner"-style per-caller state, and replies are loaded via CMSGet.
	reactions, _, err := s.loadReactions(ctx, commentIDs(flat), 0)
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	out := make([]comment_dto.Response, len(flat))
	for i, c := range flat {
		out[i] = toResponse(c, reactions, nil, 0, nil)
	}
	return out, int(total), nil
}

func (s *ServiceImpl) CMSGet(ctx context.Context, id, currentUserID int64) (comment_dto.Response, error) {
	return s.getWithThread(ctx, id, currentUserID)
}

func (s *ServiceImpl) BulkDelete(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return apperror.BadRequest("Tidak ada komentar yang dipilih")
	}
	var allIDs []int64
	for _, id := range ids {
		descendants, err := s.repo.CollectDescendantIDs(ctx, id)
		if err != nil {
			continue // already gone, e.g. removed as a descendant of another id earlier in this batch
		}
		allIDs = append(allIDs, descendants...)
	}
	mediaPaths, _ := s.repo.MediaPathsByIDs(ctx, allIDs)
	for _, id := range ids {
		_ = s.repo.Delete(ctx, id) // best-effort; already-cascaded ids are not an error
	}
	for _, p := range mediaPaths {
		_ = s.upload.DeleteFile(p)
	}
	return nil
}

// DeleteByContent implements the CommentCleaner contract expected by
// article_service/news_service (techspec §8.4).
func (s *ServiceImpl) DeleteByContent(ctx context.Context, contentType string, contentID int64) error {
	ids, err := s.repo.IDsByContent(ctx, contentType, contentID)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	mediaPaths, _ := s.repo.MediaPathsByIDs(ctx, ids)
	if err := s.repo.DeleteByContent(ctx, contentType, contentID); err != nil {
		return err
	}
	for _, p := range mediaPaths {
		_ = s.upload.DeleteFile(p)
	}
	return nil
}

// getWithThread fetches one comment plus its full reply subtree (however
// deep it goes below id) and returns it as a single Response node. Reused
// by Create/Update (fresh comment, empty subtree) and CMSGet (admin detail).
func (s *ServiceImpl) getWithThread(ctx context.Context, id, currentUserID int64) (comment_dto.Response, error) {
	target, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return comment_dto.Response{}, apperror.NotFound("Komentar tidak ditemukan")
	}
	ids, err := s.repo.CollectDescendantIDs(ctx, id)
	if err != nil {
		return comment_dto.Response{}, apperror.Internal("")
	}
	flat, err := s.repo.FindByIDs(ctx, ids)
	if err != nil {
		return comment_dto.Response{}, apperror.Internal("")
	}
	reactions, userReactions, err := s.loadReactions(ctx, ids, currentUserID)
	if err != nil {
		return comment_dto.Response{}, apperror.Internal("")
	}
	for _, node := range buildTree(flat, target.ParentID, reactions, userReactions, currentUserID) {
		if node.CommentID == id {
			return node, nil
		}
	}
	return comment_dto.Response{}, apperror.NotFound("Komentar tidak ditemukan")
}

func (s *ServiceImpl) loadReactions(ctx context.Context, ids []int64, currentUserID int64) (map[int64]map[string]int64, map[int64][]string, error) {
	counts, err := s.repo.ReactionCounts(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	userTypes, err := s.repo.UserReactionTypes(ctx, ids, currentUserID)
	if err != nil {
		return nil, nil, err
	}
	return counts, userTypes, nil
}

func commentIDs(comments []comment_model.Comment) []int64 {
	ids := make([]int64, len(comments))
	for i, c := range comments {
		ids[i] = c.CommentID
	}
	return ids
}

// buildTree recursively groups a flat comment list by parentID into the
// nested Response shape. parentID nil selects top-level comments.
func buildTree(flat []comment_model.Comment, parentID *int64, reactions map[int64]map[string]int64, userReactions map[int64][]string, currentUserID int64) []comment_dto.Response {
	var out []comment_dto.Response
	for _, c := range flat {
		if !samePtr(c.ParentID, parentID) {
			continue
		}
		childID := c.CommentID
		replies := buildTree(flat, &childID, reactions, userReactions, currentUserID)
		out = append(out, toResponse(c, reactions, userReactions, currentUserID, replies))
	}
	return out
}

func samePtr(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func toResponse(c comment_model.Comment, reactions map[int64]map[string]int64, userReactions map[int64][]string, currentUserID int64, replies []comment_dto.Response) comment_dto.Response {
	counts := reactions[c.CommentID]
	if counts == nil {
		counts = map[string]int64{}
	}
	// userTypes must never serialize as JSON null: the frontend calls
	// .includes() on it unconditionally, which throws on null and breaks
	// mid-render for any comment someone else reacted to (a nil-map lookup —
	// including the case where userReactions itself is nil, as CMSList
	// passes — returns Go's nil slice, not an empty one).
	userTypes := userReactions[c.CommentID]
	if userTypes == nil {
		userTypes = []string{}
	}
	if replies == nil {
		replies = []comment_dto.Response{}
	}
	return comment_dto.Response{
		CommentID:   c.CommentID,
		ContentType: c.ContentType,
		ContentID:   c.ContentID,
		CommentText: strOr(c.CommentText),
		MediaURL:    strOr(c.MediaURL),
		MediaType:   strOr(c.MediaType),
		ParentID:    c.ParentID,
		IsOwner:     currentUserID != 0 && c.CreatedBy == currentUserID,
		CreatedDate: c.CreatedDate.Format("2006-01-02 15:04:05"),
		Author:      comment_dto.AuthorDTO{UserID: c.CreatedBy, Name: c.AuthorName, Photo: strOr(c.AuthorPhoto)},
		Reactions:   comment_dto.ReactionsDTO{Counts: counts, UserTypes: userTypes},
		Replies:     replies,
	}
}

func strOr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ---- GIPHY proxy (GifSearch / GifCategories) ----

type giphyImage struct {
	URL string `json:"url"`
}

type giphyImages struct {
	Original         giphyImage `json:"original"`
	FixedHeightSmall giphyImage `json:"fixed_height_small"`
	Downsized        giphyImage `json:"downsized"`
}

type giphyItem struct {
	ID     string      `json:"id"`
	Title  string      `json:"title"`
	Images giphyImages `json:"images"`
}

type giphyListResponse struct {
	Data []giphyItem `json:"data"`
}

type giphyCategory struct {
	Name        string `json:"name"`
	NameEncoded string `json:"name_encoded"`
}

type giphyCategoriesResponse struct {
	Data []giphyCategory `json:"data"`
}

func (s *ServiceImpl) GifSearch(ctx context.Context, query, tab string) ([]comment_dto.GifItem, error) {
	if s.giphyAPIKey == "" {
		return []comment_dto.GifItem{}, nil
	}
	if tab != "stickers" {
		tab = "gifs"
	}
	endpoint := "search"
	if query == "" {
		endpoint = "trending"
	}
	reqURL := fmt.Sprintf("https://api.giphy.com/v1/%s/%s?api_key=%s&limit=24&rating=g&lang=id",
		tab, endpoint, url.QueryEscape(s.giphyAPIKey))
	if query != "" {
		reqURL += "&q=" + url.QueryEscape(query)
	}

	reqCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()

	var body giphyListResponse
	if err := getJSON(reqCtx, reqURL, &body); err != nil {
		return []comment_dto.GifItem{}, nil // GIPHY being unreachable must not block comment posting
	}

	out := make([]comment_dto.GifItem, 0, len(body.Data))
	for _, item := range body.Data {
		preview := item.Images.FixedHeightSmall.URL
		if preview == "" {
			preview = item.Images.Downsized.URL
		}
		out = append(out, comment_dto.GifItem{ID: item.ID, Preview: preview, URL: item.Images.Original.URL, Title: item.Title})
	}
	return out, nil
}

func (s *ServiceImpl) GifCategories(ctx context.Context) ([]comment_dto.GifCategory, error) {
	if s.giphyAPIKey == "" {
		return []comment_dto.GifCategory{}, nil
	}
	reqURL := fmt.Sprintf("https://api.giphy.com/v1/gifs/categories?api_key=%s", url.QueryEscape(s.giphyAPIKey))

	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var body giphyCategoriesResponse
	if err := getJSON(reqCtx, reqURL, &body); err != nil {
		return []comment_dto.GifCategory{}, nil
	}

	limit := min(len(body.Data), 8)
	out := make([]comment_dto.GifCategory, 0, limit)
	for _, c := range body.Data[:limit] {
		out = append(out, comment_dto.GifCategory{Name: c.Name, Slug: c.NameEncoded})
	}
	return out, nil
}

func getJSON(ctx context.Context, reqURL string, target interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("giphy: unexpected status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}
