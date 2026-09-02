package schedule_service

import (
	"context"
	"strings"
	"time"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/ptr"
	"fsldk-api/modules/schedule/schedule_dto"
	"fsldk-api/modules/schedule/schedule_model"
	"fsldk-api/modules/schedule/schedule_repository"
)

const (
	dateLayout      = "2006-01-02"
	stampLayout     = "2006-01-02 15:04:05"
	maxRangeDays    = 92  // public calendar window cap
	maxDurationDays = 366 // single-activity span guard
)

// sortColumns is the CMS list sort whitelist. The public list uses a fixed order.
var sortColumns = map[string]string{
	"title":       "s.title",
	"startDate":   "s.startDate",
	"category":    "s.category",
	"createdDate": "s.createdDate",
}

// ServiceImpl is the Service implementation.
type ServiceImpl struct {
	repo schedule_repository.Repository
}

// NewService creates the schedule Service.
func NewService(repo schedule_repository.Repository) Service {
	return &ServiceImpl{repo: repo}
}

func (s *ServiceImpl) PublicList(ctx context.Context, from, to string) ([]schedule_dto.Response, error) {
	fromT, ok := parseDate(from)
	if !ok {
		return nil, apperror.BadRequest("Parameter 'from' tidak valid (gunakan format YYYY-MM-DD)")
	}
	toT, ok := parseDate(to)
	if !ok {
		return nil, apperror.BadRequest("Parameter 'to' tidak valid (gunakan format YYYY-MM-DD)")
	}
	if toT.Before(fromT) {
		return nil, apperror.BadRequest("Rentang tanggal tidak valid")
	}
	if toT.Sub(fromT) > maxRangeDays*24*time.Hour {
		return nil, apperror.BadRequest("Rentang tanggal terlalu lebar (maksimal 92 hari)")
	}

	rows, _, err := s.repo.List(ctx, schedule_dto.Filter{
		ActiveOnly: true,
		DateFrom:   from,
		DateTo:     to,
		OrderBy:    "s.startDate ASC, s.isAllDay DESC, s.startTime ASC",
	})
	if err != nil {
		return nil, apperror.Internal("")
	}
	return toResponses(rows), nil
}

func (s *ServiceImpl) CMSList(ctx context.Context, q dto.ListQuery, f schedule_dto.Filter) ([]schedule_dto.Response, int, error) {
	f.ActiveOnly = false
	f.Limit = q.Limit
	f.Offset = q.Offset()
	f.OrderBy = q.OrderBy(sortColumns, "s.startDate DESC")
	if f.Search == "" {
		f.Search = q.Search
	}
	rows, total, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	return toResponses(rows), int(total), nil
}

func (s *ServiceImpl) Get(ctx context.Context, id int64) (schedule_dto.Response, error) {
	m, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return schedule_dto.Response{}, apperror.NotFound("Jadwal tidak ditemukan")
	}
	return toResponse(m), nil
}

func (s *ServiceImpl) Create(ctx context.Context, req schedule_dto.Request, actorID int64) (schedule_dto.Response, error) {
	entity, err := buildEntity(req)
	if err != nil {
		return schedule_dto.Response{}, err
	}
	id, err := s.repo.Create(ctx, entity, actorID)
	if err != nil {
		return schedule_dto.Response{}, apperror.Internal("Gagal menyimpan jadwal")
	}
	return s.Get(ctx, id)
}

func (s *ServiceImpl) Update(ctx context.Context, id int64, req schedule_dto.Request, actorID int64) (schedule_dto.Response, error) {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return schedule_dto.Response{}, apperror.NotFound("Jadwal tidak ditemukan")
	}
	entity, err := buildEntity(req)
	if err != nil {
		return schedule_dto.Response{}, err
	}
	if err := s.repo.Update(ctx, id, entity, actorID); err != nil {
		return schedule_dto.Response{}, apperror.Internal("")
	}
	return s.Get(ctx, id)
}

func (s *ServiceImpl) SetActive(ctx context.Context, id int64, isActive bool, actorID int64) error {
	err := s.repo.SetActive(ctx, id, isActive, actorID)
	if err == schedule_repository.ErrNotFound {
		return apperror.NotFound("Jadwal tidak ditemukan")
	}
	if err != nil {
		return apperror.Internal("")
	}
	return nil
}

func (s *ServiceImpl) Delete(ctx context.Context, id int64) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return apperror.NotFound("Jadwal tidak ditemukan")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return apperror.Internal("")
	}
	return nil
}

// buildEntity validates the request's cross-field rules (tech spec §4.3),
// normalises it (trim strings, drop endDate == startDate, null out times when
// all-day), and maps it to a Schedule model.
func buildEntity(req schedule_dto.Request) (schedule_model.Schedule, error) {
	start, ok := parseDate(req.StartDate)
	if !ok {
		return schedule_model.Schedule{}, apperror.BadRequest("Format tanggal mulai tidak valid (gunakan format YYYY-MM-DD)")
	}

	var endPtr *time.Time
	if strings.TrimSpace(req.EndDate) != "" {
		end, ok := parseDate(req.EndDate)
		if !ok {
			return schedule_model.Schedule{}, apperror.BadRequest("Format tanggal selesai tidak valid (gunakan format YYYY-MM-DD)")
		}
		if end.Before(start) {
			return schedule_model.Schedule{}, apperror.BadRequest("Tanggal selesai tidak boleh sebelum tanggal mulai")
		}
		if end.Sub(start) > maxDurationDays*24*time.Hour {
			return schedule_model.Schedule{}, apperror.BadRequest("Rentang tanggal kegiatan terlalu panjang")
		}
		// A single-day activity is always stored as endDate IS NULL.
		if !end.Equal(start) {
			e := end
			endPtr = &e
		}
	}

	startTime, endTime := "", ""
	if !req.IsAllDay {
		if strings.TrimSpace(req.StartTime) == "" {
			return schedule_model.Schedule{}, apperror.BadRequest("Jam mulai wajib diisi untuk kegiatan yang tidak sepanjang hari")
		}
		startMin, ok := parseClock(req.StartTime)
		if !ok {
			return schedule_model.Schedule{}, apperror.BadRequest("Format jam mulai tidak valid (gunakan format HH:mm)")
		}
		startTime = normalizeClock(req.StartTime)

		if strings.TrimSpace(req.EndTime) != "" {
			endMin, ok := parseClock(req.EndTime)
			if !ok {
				return schedule_model.Schedule{}, apperror.BadRequest("Format jam selesai tidak valid (gunakan format HH:mm)")
			}
			endTime = normalizeClock(req.EndTime)
			// Only a single-day activity compares the two clock values — for a
			// multi-day activity endTime is the clock on endDate.
			if endPtr == nil && endMin <= startMin {
				return schedule_model.Schedule{}, apperror.BadRequest("Jam selesai harus setelah jam mulai")
			}
		}
	}

	return schedule_model.Schedule{
		Title:         strings.TrimSpace(req.Title),
		Description:   ptr.Str(strings.TrimSpace(req.Description)),
		Category:      req.Category,
		StartDate:     start,
		EndDate:       endPtr,
		IsAllDay:      req.IsAllDay,
		StartTime:     ptr.Str(startTime),
		EndTime:       ptr.Str(endTime),
		Location:      ptr.Str(strings.TrimSpace(req.Location)),
		Organizer:     ptr.Str(strings.TrimSpace(req.Organizer)),
		ContactPerson: ptr.Str(strings.TrimSpace(req.ContactPerson)),
		URL:           ptr.Str(strings.TrimSpace(req.URL)),
	}, nil
}

func parseDate(v string) (time.Time, bool) {
	t, err := time.ParseInLocation(dateLayout, strings.TrimSpace(v), time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// parseClock accepts "HH:mm" or "HH:mm:ss" and returns minutes since midnight.
func parseClock(v string) (int, bool) {
	v = strings.TrimSpace(v)
	for _, layout := range []string{"15:04", "15:04:05"} {
		if t, err := time.Parse(layout, v); err == nil {
			return t.Hour()*60 + t.Minute(), true
		}
	}
	return 0, false
}

// normalizeClock trims "HH:mm:ss" down to "HH:mm".
func normalizeClock(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 5 {
		return v[:5]
	}
	return v
}

func toResponse(m schedule_model.Schedule) schedule_dto.Response {
	resp := schedule_dto.Response{
		ScheduleID:    m.ScheduleID,
		Title:         m.Title,
		Category:      m.Category,
		Description:   m.Description,
		StartDate:     m.StartDate.Format(dateLayout),
		IsAllDay:      m.IsAllDay,
		StartTime:     trimSeconds(m.StartTime),
		EndTime:       trimSeconds(m.EndTime),
		Location:      m.Location,
		Organizer:     m.Organizer,
		ContactPerson: m.ContactPerson,
		URL:           m.URL,
		IsActive:      m.IsActive,
		CreatedDate:   m.CreatedDate.Format(stampLayout),
	}
	if m.EndDate != nil {
		v := m.EndDate.Format(dateLayout)
		resp.EndDate = &v
	}
	if m.UpdatedDate != nil {
		v := m.UpdatedDate.Format(stampLayout)
		resp.UpdatedDate = &v
	}
	return resp
}

func toResponses(rows []schedule_model.Schedule) []schedule_dto.Response {
	out := make([]schedule_dto.Response, 0, len(rows))
	for _, m := range rows {
		out = append(out, toResponse(m))
	}
	return out
}

// trimSeconds turns a "HH:mm:ss" TIME column value into "HH:mm" (nil-safe).
func trimSeconds(v *string) *string {
	if v == nil {
		return nil
	}
	t := normalizeClock(*v)
	return &t
}
