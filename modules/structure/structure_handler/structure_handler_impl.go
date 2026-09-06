package structure_handler

import (
	"errors"
	"strconv"
	"strings"

	"fsldk-api/base/appctx"
	"fsldk-api/base/apperror"

	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/modules/structure/structure_dto"
	"fsldk-api/modules/structure/structure_repository"
	"fsldk-api/modules/structure/structure_service"

	"github.com/gin-gonic/gin"
)

type HandlerImpl struct{ svc structure_service.Service }

// NewHandler creates a new structure handler.
func NewHandler(svc structure_service.Service) Handler {
	return &HandlerImpl{svc: svc}
}

func (h *HandlerImpl) ListPublic(c *gin.Context) {
	// Public endpoint doesn't need pagination according to techspec
	filter := structure_dto.Filter{
		SortBy:    "createdDate",
		SortOrder: "desc",
	}

	structures, _, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		httphelper.Error(c, err)
		return
	}

	httphelper.Success(c, "Berhasil mengambil daftar struktur", structures)
}

func (h *HandlerImpl) ListCMS(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "15"))
	if limit < 1 {
		limit = 15
	}
	offset := (page - 1) * limit

	filter := structure_dto.Filter{
		Search:    strings.TrimSpace(c.Query("search")),
		SortBy:    c.DefaultQuery("sort_by", "createdDate"),
		SortOrder: c.DefaultQuery("sort_order", "desc"),
		Limit:     limit,
		Offset:    offset,
	}

	structures, total, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		httphelper.Error(c, err)
		return
	}

	res := httphelper.BuildPagination(c, structures, int(total), page, limit)
	httphelper.Success(c, "Berhasil mengambil daftar struktur", res)
}

func (h *HandlerImpl) ShowCMS(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return
	}

	s, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, structure_repository.ErrNotFound) {
			httphelper.Error(c, apperror.NotFound("Struktur tidak ditemukan"))
			return
		}
		httphelper.Error(c, err)
		return
	}

	httphelper.Success(c, "Berhasil mengambil struktur", s)
}

func (h *HandlerImpl) Create(c *gin.Context) {
	var req structure_dto.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Request tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}

	authorID := appctx.UserID(c)
	id, err := h.svc.Create(c.Request.Context(), req, authorID)
	if err != nil {
		httphelper.Error(c, err)
		return
	}

	httphelper.Created(c, "Struktur berhasil dibuat", gin.H{"structureID": id})
}

func (h *HandlerImpl) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return
	}

	var req structure_dto.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Request tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}

	updatedBy := appctx.UserID(c)
	err = h.svc.Update(c.Request.Context(), id, req, updatedBy)
	if err != nil {
		if errors.Is(err, structure_repository.ErrNotFound) {
			httphelper.Error(c, apperror.NotFound("Struktur tidak ditemukan"))
			return
		}
		httphelper.Error(c, err)
		return
	}

	httphelper.Success(c, "Struktur berhasil diupdate", nil)
}

func (h *HandlerImpl) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return
	}

	err = h.svc.Delete(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, structure_repository.ErrNotFound) {
			httphelper.Error(c, apperror.NotFound("Struktur tidak ditemukan"))
			return
		}
		httphelper.Error(c, err)
		return
	}

	httphelper.Success(c, "Struktur berhasil dihapus", nil)
}
