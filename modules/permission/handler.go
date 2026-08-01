package permission

import (
	"fsldk-api/base/apperror"
	"fsldk-api/base/appctx"
	"fsldk-api/base/httphelper"
	"fsldk-api/constants"
	"fsldk-api/middlewares"

	"github.com/gin-gonic/gin"
)

// Handler menangani request HTTP terkait permission & menu.
type Handler struct{ svc *Service }

// NewHandler membuat Handler permission.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// ListAll menangani GET /permissions.
func (h *Handler) ListAll(c *gin.Context) {
	data, err := h.svc.ListAll(c.Request.Context())
	if err != nil {
		httphelper.Error(c, apperror.Internal(""))
		return
	}
	httphelper.Success(c, "", data)
}

// Menu menangani GET /me/menus — daftar menu sidebar CMS sesuai permission role.
func (h *Handler) Menu(c *gin.Context) {
	data, err := h.svc.Menu(c.Request.Context(), appctx.RoleID(c))
	if err != nil {
		httphelper.Error(c, apperror.Internal(""))
		return
	}
	httphelper.Success(c, "", data)
}

// RegisterRoutes mendaftarkan endpoint permission & menu.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, mw *middlewares.Middleware) {
	// Daftar seluruh permission (untuk halaman manajemen role).
	rg.GET("/permissions", mw.Auth(), mw.RequireVerified(), mw.RequirePermission(constants.PermRoleView), h.ListAll)

	// Menu sidebar CMS: cukup terautentikasi + terverifikasi (tanpa permission khusus).
	rg.GET("/me/menus", mw.Auth(), mw.RequireVerified(), h.Menu)
}
