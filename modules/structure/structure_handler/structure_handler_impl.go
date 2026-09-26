package structure_handler

import (
	"errors"
	"strconv"
	"strings"

	"fsldk-api/base/appctx"
	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"

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
		OrderBy: "createdDate DESC",
	}

	structures, _, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		httphelper.Error(c, err)
		return
	}

	httphelper.Success(c, "Berhasil mengambil daftar struktur", structures)
}

func (h *HandlerImpl) ListCMS(c *gin.Context) {
	q := dto.ParseListQuery(c)
	filter := structure_dto.Filter{
		DateFrom: strings.TrimSpace(c.Query("dateFrom")),
		DateTo:   strings.TrimSpace(c.Query("dateTo")),
	}

	structures, total, err := h.svc.CMSList(c.Request.Context(), q, filter)
	if err != nil {
		httphelper.Error(c, err)
		return
	}

	res := httphelper.BuildPagination(c, structures, int(total), q.Page, q.Limit)
	httphelper.Success(c, "Berhasil mengambil daftar struktur", res)
}

func (h *HandlerImpl) BulkDelete(c *gin.Context) {
	var req structure_dto.BulkDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	if err := h.svc.BulkDelete(c.Request.Context(), req.IDs); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Struktur terpilih berhasil dihapus", nil)
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
