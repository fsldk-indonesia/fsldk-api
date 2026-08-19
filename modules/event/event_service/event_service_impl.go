package event_service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/ptr"
	"fsldk-api/base/slug"
	"fsldk-api/modules/event/event_dto"
	"fsldk-api/modules/event/event_model"
	"fsldk-api/modules/event/event_repository"
)

// allowedSortCols maps public sort values to the Filter.SortBy strings.
var allowedSortCols = map[string]string{
	"title":       "title",
	"newest":      "newest",
	"startDate":   "startDate",
	"createdDate": "createdDate",
}

// ServiceImpl is the concrete implementation of Service.
type ServiceImpl struct {
	repo    event_repository.Repository
	comment CommentCleaner
}

// NewService constructs a Service backed by the given Repository.
func NewService(repo event_repository.Repository, comment CommentCleaner) Service {
	return &ServiceImpl{repo: repo, comment: comment}
}

// --- Public API ---

func (s *ServiceImpl) PublicList(ctx context.Context, q dto.ListQuery, divisions, years, statuses []string, sort string) ([]event_dto.EventResponse, int, error) {
	sortBy := "newest"
	if sort == "title" {
		sortBy = "title"
	}
	events, total, err := s.repo.List(ctx, event_dto.Filter{
		Search:        q.Search,
		Divisions:     divisions,
		Years:         years,
		Statuses:      statuses,
		PublishedOnly: true,
		SortBy:        sortBy,
		SortOrder:     "DESC",
		Limit:         q.Limit,
		Offset:        q.Offset(),
	})
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	return s.toResponses(events), int(total), nil
}

func (s *ServiceImpl) PublicDetail(ctx context.Context, slug string) (event_dto.EventResponse, error) {
	e, err := s.repo.FindBySlug(ctx, slug)
	if err != nil || !e.IsPublished {
		return event_dto.EventResponse{}, apperror.NotFound("Event tidak ditemukan")
	}
	// Increment view count in the background — errors are intentionally ignored.
	go func() { _ = s.repo.IncrementViewCount(context.Background(), e.EventID) }()
	e.ViewCount++ // optimistic local update for the current response
	return s.toResponse(e), nil
}

// --- CMS API ---

func (s *ServiceImpl) CMSList(ctx context.Context, q dto.ListQuery, division string) ([]event_model.Event, int, error) {
	// Parse sort direction from the Sort field (e.g. "-createdDate" → DESC)
	sortBy, sortOrder := parseSortQuery(q.Sort)
	divisions := []string{}
	if division != "" {
		divisions = []string{division}
	}
	events, total, err := s.repo.List(ctx, event_dto.Filter{
		Search:    q.Search,
		Divisions: divisions,
		SortBy:    sortBy,
		SortOrder: sortOrder,
		Limit:     q.Limit,
		Offset:    q.Offset(),
	})
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	if events == nil {
		events = []event_model.Event{}
	}
	return events, int(total), nil
}

func (s *ServiceImpl) CMSGet(ctx context.Context, id int64) (event_model.Event, error) {
	e, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return event_model.Event{}, apperror.NotFound("Event tidak ditemukan")
	}
	return e, nil
}

func (s *ServiceImpl) Create(ctx context.Context, req event_dto.CreateRequest, authorID int64) (event_model.Event, error) {
	entity, err := s.fromRequest(req)
	if err != nil {
		return event_model.Event{}, err
	}
	slugStr, err := s.uniqueSlug(ctx, req.EventTitle, 0)
	if err != nil {
		return event_model.Event{}, err
	}
	entity.EventSlug = slugStr
	id, err := s.repo.Create(ctx, entity, authorID)
	if err != nil {
		return event_model.Event{}, apperror.Internal("Gagal menyimpan event")
	}
	return s.CMSGet(ctx, id)
}

func (s *ServiceImpl) Update(ctx context.Context, id int64, req event_dto.UpdateRequest, updatedBy int64) (event_model.Event, error) {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return event_model.Event{}, apperror.NotFound("Event tidak ditemukan")
	}
	entity, err := s.fromRequest(req)
	if err != nil {
		return event_model.Event{}, err
	}
	// Re-slug only when the title has changed.
	slugStr := existing.EventSlug
	if existing.EventTitle != req.EventTitle {
		slugStr, err = s.uniqueSlug(ctx, req.EventTitle, id)
		if err != nil {
			return event_model.Event{}, err
		}
	}
	entity.EventSlug = slugStr
	if err := s.repo.Update(ctx, id, entity, updatedBy); err != nil {
		return event_model.Event{}, apperror.Internal("")
	}
	return s.CMSGet(ctx, id)
}

func (s *ServiceImpl) Delete(ctx context.Context, id int64) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return apperror.NotFound("Event tidak ditemukan")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return apperror.Internal("")
	}
	// Best-effort: ms_comment has no FK to ms_event (see comment techspec
	// §3.1a), so comments aren't cascaded by the database — clean them up
	// explicitly. A failure here does not roll back the event delete.
	_ = s.comment.DeleteByContent(ctx, "event", id)
	return nil
}

// --- Helpers ---

// fromRequest maps a CreateRequest to an Event model, parsing optional datetime strings.
func (s *ServiceImpl) fromRequest(req event_dto.CreateRequest) (event_model.Event, error) {
	e := event_model.Event{
		EventTitle:       req.EventTitle,
		EventDivision:    req.EventDivision,
		EventContent:     req.EventContent,
		EventImage:       ptr.Str(req.EventImage),
		Location:         ptr.Str(req.Location),
		Place:            ptr.Str(req.Place),
		LocationLink:     ptr.Str(req.LocationLink),
		RegistrationLink: ptr.Str(req.RegistrationLink),
		DocumentLink:     ptr.Str(req.DocumentLink),
		PresentationLink: ptr.Str(req.PresentationLink),
		ContactPerson1:   ptr.Str(req.ContactPerson1),
		NameCp1:          ptr.Str(req.NameCp1),
		ContactPerson2:   ptr.Str(req.ContactPerson2),
		NameCp2:          ptr.Str(req.NameCp2),
		Tag:              ptr.Str(req.Tag),
		IsPublished:      req.IsPublished,
	}
	var err error
	if e.StartDate, err = parseDateTime(req.StartDate); err != nil {
		return event_model.Event{}, apperror.BadRequest("Format startDate tidak valid (gunakan ISO8601)")
	}
	if e.EndDate, err = parseDateTime(req.EndDate); err != nil {
		return event_model.Event{}, apperror.BadRequest("Format endDate tidak valid (gunakan ISO8601)")
	}
	if e.CloseRegistDate, err = parseDateTime(req.CloseRegistDate); err != nil {
		return event_model.Event{}, apperror.BadRequest("Format closeRegistDate tidak valid (gunakan ISO8601)")
	}
	return e, nil
}

// parseDateTime parses an ISO8601 datetime string; returns nil when the string is empty.
func parseDateTime(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.ParseInLocation(f, s, time.Local); err == nil {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("cannot parse datetime: %q", s)
}

// parseSortQuery splits the dto.ListQuery.Sort field (e.g. "-createdDate") into column + direction.
func parseSortQuery(sort string) (col, dir string) {
	sort = strings.TrimSpace(sort)
	dir = "DESC"
	if strings.HasPrefix(sort, "-") {
		sort = strings.TrimPrefix(sort, "-")
	} else if sort != "" {
		dir = "ASC"
	}
	allowed, ok := allowedSortCols[sort]
	if !ok {
		return "newest", "DESC"
	}
	return allowed, dir
}

// uniqueSlug generates a URL-safe slug from title, appending a counter if needed.
func (s *ServiceImpl) uniqueSlug(ctx context.Context, title string, exceptID int64) (string, error) {
	base := slug.Make(title)
	candidate := base
	for i := 2; i < 100; i++ {
		exists, err := s.repo.SlugExists(ctx, candidate, exceptID)
		if err != nil {
			return "", apperror.Internal("")
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
	return fmt.Sprintf("%s-%d", base, exceptID), nil
}

// computeStatus derives event status from current time vs start/end dates.
func computeStatus(e event_model.Event) string {
	now := time.Now()
	if e.StartDate != nil && now.Before(*e.StartDate) {
		return string(event_model.StatusUpcoming)
	}
	if e.StartDate != nil && e.EndDate != nil &&
		(now.Equal(*e.StartDate) || now.After(*e.StartDate)) &&
		(now.Equal(*e.EndDate) || now.Before(*e.EndDate)) {
		return string(event_model.StatusOngoing)
	}
	return string(event_model.StatusPast)
}

// computeRegistOpen returns true when registration is still open.
func computeRegistOpen(e event_model.Event, status string) bool {
	if status == string(event_model.StatusPast) {
		return false
	}
	if e.CloseRegistDate == nil {
		return true
	}
	return time.Now().Before(*e.CloseRegistDate)
}

func (s *ServiceImpl) toResponse(e event_model.Event) event_dto.EventResponse {
	status := computeStatus(e)
	registOpen := computeRegistOpen(e, status)
	return event_dto.EventResponse{
		EventID: e.EventID, EventTitle: e.EventTitle, EventSlug: e.EventSlug,
		EventDivision: e.EventDivision, EventContent: e.EventContent, EventImage: e.EventImage,
		StartDate: e.StartDate, EndDate: e.EndDate, CloseRegistDate: e.CloseRegistDate,
		Location: e.Location, Place: e.Place, LocationLink: e.LocationLink,
		RegistrationLink: e.RegistrationLink, DocumentLink: e.DocumentLink,
		PresentationLink: e.PresentationLink,
		ContactPerson1:   e.ContactPerson1, NameCp1: e.NameCp1,
		ContactPerson2: e.ContactPerson2, NameCp2: e.NameCp2,
		Tag: e.Tag, IsPublished: e.IsPublished, ViewCount: e.ViewCount,
		Status: status, RegistOpen: registOpen, AuthorID: e.AuthorID,
	}
}

func (s *ServiceImpl) toResponses(events []event_model.Event) []event_dto.EventResponse {
	out := make([]event_dto.EventResponse, 0, len(events))
	for _, e := range events {
		out = append(out, s.toResponse(e))
	}
	return out
}
