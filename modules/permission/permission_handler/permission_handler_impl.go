package permission_handler

import (
	"fsldk-api/base/apperror"
	"fsldk-api/base/appctx"
	"fsldk-api/base/httphelper"
	"fsldk-api/modules/permission/permission_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct{ svc permission_service.Service }

// NewHandler membuat Handler permission.
func NewHandler(svc permission_service.Service) Handler { return &HandlerImpl{svc: svc} }

// ListAll menangani GET /permissions.
func (h *HandlerImpl) ListAll(c *gin.Context) {
	data, err := h.svc.ListAll(c.Request.Context())
	if err != nil {
		httphelper.Error(c, apperror.Internal(""))
		return
	}
	httphelper.Success(c, "", data)
}

// Menu menangani GET /me/menus — daftar menu sidebar CMS sesuai permission role.
func (h *HandlerImpl) Menu(c *gin.Context) {
	data, err := h.svc.Menu(c.Request.Context(), appctx.RoleID(c))
	if err != nil {
		httphelper.Error(c, apperror.Internal(""))
		return
	}
	httphelper.Success(c, "", data)
}
