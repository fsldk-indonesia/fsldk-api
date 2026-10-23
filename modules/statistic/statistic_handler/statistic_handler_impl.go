package statistic_handler

import (
	"strconv"
	"strings"

	"fsldk-api/base/dto"
	"fsldk-api/base/httphelper"
	"fsldk-api/modules/statistic/statistic_dto"
	"fsldk-api/modules/statistic/statistic_service"

	"github.com/gin-gonic/gin"
)

type handlerImpl struct {
	svc statistic_service.Service
}

// directoryMaxLimit is Directory()'s own, higher result-size ceiling — the
// public "Direktori Jaringan" renders ALL matching organizations at once as
// one Puskomnas -> Puskomda -> LDK tree (no numbered pagination UI, see
// statistic.index.page.ts), unlike every other list endpoint. The shared
// 100-row clamp in base/dto/pagination.go (ParseListQuery) is deliberately
// NOT raised there — it's used across many unrelated modules — so this
// endpoint overrides it locally instead. 2000 comfortably covers the current
// 590 active organizations (2026-10) with headroom for years of growth.
const directoryMaxLimit = 2000

// NewHandler creates a new instance of statistic Handler.
func NewHandler(svc statistic_service.Service) Handler {
	return &handlerImpl{svc: svc}
}

// trimmedNonEmpty trims each value and drops blanks — multi-select filter
// params arrive as repeated query keys (?type=LDK&type=PUSKOMDA, parsed via
// c.QueryArray), not comma-joined, mirroring the same convention used by the
// Galeri/Artikel modules' public filters (article_handler_impl.go).
func trimmedNonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// parseInt64s best-effort parses each value as an int64, skipping invalid ones.
func parseInt64s(values []string) []int64 {
	out := make([]int64, 0, len(values))
	for _, v := range values {
		if n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
			out = append(out, n)
		}
	}
	return out
}

func (h *handlerImpl) NetworkStats(c *gin.Context) {
	resp, err := h.svc.NetworkStats(c.Request.Context())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Berhasil mengambil statistik jaringan", resp)
}

func (h *handlerImpl) Directory(c *gin.Context) {
	q := dto.ParseListQuery(c)
	// q.Limit is already clamped to 100 by ParseListQuery — re-read the raw
	// query value here to apply this endpoint's own higher ceiling instead
	// (see directoryMaxLimit comment above).
	if raw := c.Query("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > q.Limit {
			if n > directoryMaxLimit {
				n = directoryMaxLimit
			}
			q.Limit = n
		}
	}
	f := statistic_dto.DirectoryFilter{
		OrganizationTypeCodes: trimmedNonEmpty(c.QueryArray("type")),
		ProvinceNames:         trimmedNonEmpty(c.QueryArray("province")),
		CityNames:             trimmedNonEmpty(c.QueryArray("city")),
		ParentOrganizationIDs: parseInt64s(c.QueryArray("puskomda")),
		LevelCodes:            trimmedNonEmpty(c.QueryArray("level")),
	}
	items, total, err := h.svc.Directory(c.Request.Context(), q, f)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Berhasil mengambil direktori jaringan", httphelper.BuildPagination(c, items, total, q.Page, q.Limit))
}
