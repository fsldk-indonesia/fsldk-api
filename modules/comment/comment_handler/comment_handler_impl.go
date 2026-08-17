package comment_handler

import (
	"strconv"
	"strings"

	"fsldk-api/base/appctx"
	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/constants"
	"fsldk-api/modules/comment/comment_dto"
	"fsldk-api/modules/comment/comment_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct{ svc comment_service.Service }

// NewHandler membuat Handler komentar.
func NewHandler(svc comment_service.Service) Handler { return &HandlerImpl{svc: svc} }

func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return id, true
}

// hasPermission reads the permission list stashed in context by
// middlewares.RequirePermission or middlewares.LoadPermissions. On routes
// with neither attached this always returns false, which is the correct
// fallback (owner-only).
func hasPermission(c *gin.Context, code string) bool {
	if perms, ok := c.Get(constants.CtxPermissions); ok {
		if list, ok := perms.([]string); ok {
			for _, p := range list {
				if p == code {
					return true
				}
			}
		}
	}
	return false
}

func bind[T any](c *gin.Context) (T, bool) {
	var req T
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return req, false
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return req, false
	}
	return req, true
}

func (h *HandlerImpl) PublicList(c *gin.Context) {
	contentType := c.Query("contentType")
	contentID, _ := strconv.ParseInt(c.Query("contentID"), 10, 64)
	if contentType == "" || contentID <= 0 {
		httphelper.Error(c, apperror.BadRequest("contentType dan contentID wajib diisi"))
		return
	}
	data, err := h.svc.PublicList(c.Request.Context(), contentType, contentID, appctx.UserID(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) Create(c *gin.Context) {
	req, ok := bind[comment_dto.CreateRequest](c)
	if !ok {
		return
	}
	data, err := h.svc.Create(c.Request.Context(), req, appctx.UserID(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Created(c, "Komentar berhasil dikirim", data)
}

func (h *HandlerImpl) Update(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	req, ok := bind[comment_dto.UpdateRequest](c)
	if !ok {
		return
	}
	data, err := h.svc.Update(c.Request.Context(), id, req, appctx.UserID(c), hasPermission(c, constants.PermCommentUpdate))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Komentar berhasil diperbarui", data)
}

func (h *HandlerImpl) Delete(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id, appctx.UserID(c), hasPermission(c, constants.PermCommentDelete)); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Komentar berhasil dihapus", nil)
}

func (h *HandlerImpl) React(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	req, ok := bind[comment_dto.ReactRequest](c)
	if !ok {
		return
	}
	data, err := h.svc.React(c.Request.Context(), id, appctx.UserID(c), req.ReactionType)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) GifSearch(c *gin.Context) {
	tab := c.DefaultQuery("tab", "gifs")
	data, err := h.svc.GifSearch(c.Request.Context(), strings.TrimSpace(c.Query("q")), tab)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) GifCategories(c *gin.Context) {
	data, err := h.svc.GifCategories(c.Request.Context())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) CMSList(c *gin.Context) {
	q := dto.ParseListQuery(c)
	data, total, err := h.svc.CMSList(c.Request.Context(), q, c.Query("contentType"))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", httphelper.BuildPagination(c, data, total, q.Page, q.Limit))
}

func (h *HandlerImpl) CMSGet(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	data, err := h.svc.CMSGet(c.Request.Context(), id, appctx.UserID(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) BulkDelete(c *gin.Context) {
	req, ok := bind[comment_dto.BulkDeleteRequest](c)
	if !ok {
		return
	}
	if err := h.svc.BulkDelete(c.Request.Context(), req.IDs); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Komentar berhasil dihapus", nil)
}
