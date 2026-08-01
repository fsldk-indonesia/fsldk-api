package user

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes mendaftarkan endpoint modul user (seluruhnya terproteksi
// autentikasi + verifikasi email + permission).
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, mw *middlewares.Middleware) {
	g := rg.Group("/users")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("", mw.RequirePermission(constants.PermUserView), h.List)
		g.GET("/:id", mw.RequirePermission(constants.PermUserView), h.Get)
		g.POST("", mw.RequirePermission(constants.PermUserCreate), h.Create)
		g.PUT("/:id", mw.RequirePermission(constants.PermUserUpdate), h.Update)
		g.PATCH("/:id/status", mw.RequirePermission(constants.PermUserUpdate), h.SetStatus)
		g.POST("/:id/reset-password", mw.RequirePermission(constants.PermUserUpdate), h.ResetPassword)
		g.DELETE("/:id", mw.RequirePermission(constants.PermUserDelete), h.Delete)
	}
}
