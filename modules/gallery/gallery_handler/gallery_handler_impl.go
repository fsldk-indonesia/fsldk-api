package gallery_handler

import (
	"errors"
	"strconv"
	"strings"

	"fsldk-api/base/appctx"
	"fsldk-api/base/apperror"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/modules/gallery/gallery_dto"
	"fsldk-api/modules/gallery/gallery_repository"
	"fsldk-api/modules/gallery/gallery_service"

	"github.com/gin-gonic/gin"
)

type handlerImpl struct {
	svc gallery_service.Service
}

// NewHandler creates a new gallery handler instance.
func NewHandler(svc gallery_service.Service) Handler {
	return &handlerImpl{svc: svc}
}

func (h *handlerImpl) ListPublic(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "9"))
	sort := c.DefaultQuery("sort", "newest")

	items, total, totalPages, err := h.svc.ListPublic(c.Request.Context(), page, limit, sort)
	if err != nil {
		httphelper.Error(c, err)
		return
	}

	result := gin.H{
		"data":       items,
		"page":       page,
		"limit":      limit,
		"total":      total,
		"totalPages": totalPages,
	}
	httphelper.Success(c, "Berhasil mengambil daftar galeri", result)
}

func (h *handlerImpl) ShowPublic(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return
	}

	gallery, err := h.svc.GetPublic(c.Request.Context(), id)
	if err != nil {
		httphelper.Error(c, err)
		return
	}

	httphelper.Success(c, "Berhasil mengambil detail galeri", gallery)
}

func (h *handlerImpl) ListPhotosPublic(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "12"))

	photos, err := h.svc.ListPhotosPublic(c.Request.Context(), id, page, limit)
	if err != nil {
		httphelper.Error(c, err)
		return
	}

	httphelper.Success(c, "Berhasil mengambil foto galeri", photos)
}

func (h *handlerImpl) ListCMS(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "15"))
	if limit < 1 {
		limit = 15
	}
	offset := (page - 1) * limit

	filter := gallery_dto.Filter{
		Search:     strings.TrimSpace(c.Query("search")),
		EventName:  strings.TrimSpace(c.Query("eventName")),
		EventTheme: strings.TrimSpace(c.Query("eventTheme")),
		SortBy:     c.DefaultQuery("sort_by", "createdDate"),
		SortOrder:  c.DefaultQuery("sort_order", "desc"),
		Limit:      limit,
		Offset:     offset,
	}

	galleries, total, err := h.svc.ListCMS(c.Request.Context(), filter)
	if err != nil {
		httphelper.Error(c, err)
		return
	}

	res := httphelper.BuildPagination(c, galleries, int(total), page, limit)
	httphelper.Success(c, "Berhasil mengambil daftar galeri", res)
}

func (h *handlerImpl) ShowCMS(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return
	}

	gallery, err := h.svc.GetCMS(c.Request.Context(), id)
	if err != nil {
		httphelper.Error(c, err)
		return
	}

	httphelper.Success(c, "Berhasil mengambil galeri", gallery)
}

func (h *handlerImpl) Create(c *gin.Context) {
	var req gallery_dto.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Request tidak valid: "+err.Error()))
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

	httphelper.Created(c, "Galeri berhasil dibuat", gin.H{"galleryID": id})
}

func (h *handlerImpl) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return
	}

	var req gallery_dto.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Request tidak valid: "+err.Error()))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}

	updatedBy := appctx.UserID(c)
	if err := h.svc.Update(c.Request.Context(), id, req, updatedBy); err != nil {
		httphelper.Error(c, err)
		return
	}

	httphelper.Success(c, "Galeri berhasil diperbarui", nil)
}

func (h *handlerImpl) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		httphelper.Error(c, err)
		return
	}

	httphelper.Success(c, "Galeri berhasil dihapus", nil)
}

func (h *handlerImpl) ListPhotosCMS(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	photos, err := h.svc.ListPhotosCMS(c.Request.Context(), id, page, limit)
	if err != nil {
		httphelper.Error(c, err)
		return
	}

	httphelper.Success(c, "Berhasil mengambil daftar foto", photos)
}

func (h *handlerImpl) AddPhoto(c *gin.Context) {
	galleryID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httphelper.Error(c, apperror.BadRequest("ID galeri tidak valid"))
		return
	}

	var req gallery_dto.AddPhotoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Request tidak valid: "+err.Error()))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}

	authorID := appctx.UserID(c)
	photo, err := h.svc.AddPhoto(c.Request.Context(), galleryID, req, authorID)
	if err != nil {
		httphelper.Error(c, err)
		return
	}

	httphelper.Created(c, "Foto berhasil ditambahkan", photo)
}

func (h *handlerImpl) UpdatePhoto(c *gin.Context) {
	galleryID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httphelper.Error(c, apperror.BadRequest("ID galeri tidak valid"))
		return
	}

	photoID, err := strconv.ParseInt(c.Param("photoID"), 10, 64)
	if err != nil {
		httphelper.Error(c, apperror.BadRequest("ID foto tidak valid"))
		return
	}

	var req gallery_dto.UpdatePhotoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Request tidak valid: "+err.Error()))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}

	if err := h.svc.UpdatePhoto(c.Request.Context(), galleryID, photoID, req); err != nil {
		if errors.Is(err, gallery_repository.ErrPhotoNotFound) {
			httphelper.Error(c, apperror.NotFound("Foto tidak ditemukan"))
			return
		}
		httphelper.Error(c, err)
		return
	}

	httphelper.Success(c, "Foto berhasil diperbarui", nil)
}

func (h *handlerImpl) DeletePhoto(c *gin.Context) {
	galleryID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httphelper.Error(c, apperror.BadRequest("ID galeri tidak valid"))
		return
	}

	photoID, err := strconv.ParseInt(c.Param("photoID"), 10, 64)
	if err != nil {
		httphelper.Error(c, apperror.BadRequest("ID foto tidak valid"))
		return
	}

	if err := h.svc.DeletePhoto(c.Request.Context(), galleryID, photoID); err != nil {
		if errors.Is(err, gallery_repository.ErrPhotoNotFound) {
			httphelper.Error(c, apperror.NotFound("Foto tidak ditemukan"))
			return
		}
		httphelper.Error(c, err)
		return
	}

	httphelper.Success(c, "Foto berhasil dihapus", nil)
}

func (h *handlerImpl) ReorderPhotos(c *gin.Context) {
	galleryID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httphelper.Error(c, apperror.BadRequest("ID galeri tidak valid"))
		return
	}

	var req gallery_dto.ReorderPhotosRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Request tidak valid: "+err.Error()))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}

	if err := h.svc.ReorderPhotos(c.Request.Context(), galleryID, req); err != nil {
		httphelper.Error(c, err)
		return
	}

	httphelper.Success(c, "Urutan foto berhasil diperbarui", nil)
}
