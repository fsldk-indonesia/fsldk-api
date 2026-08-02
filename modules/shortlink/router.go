// Package shortlink merangkai routing modul shortlink.
package shortlink

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/shortlink/shortlink_handler"

	"github.com/gin-gonic/gin"
)

// RegisterCMSRoutes mendaftarkan endpoint manajemen shortlink (terproteksi
// auth + verifikasi + permission), di bawah grup /api/v1.
func RegisterCMSRoutes(rg *gin.RouterGroup, h shortlink_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/shortlinks")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("", mw.RequirePermission(constants.PermShortlinkView), h.List)
		g.GET("/:id", mw.RequirePermission(constants.PermShortlinkView), h.Get)
		g.POST("", mw.RequirePermission(constants.PermShortlinkCreate), h.Create)
		g.PUT("/:id", mw.RequirePermission(constants.PermShortlinkUpdate), h.Update)
		g.DELETE("/:id", mw.RequirePermission(constants.PermShortlinkDelete), h.Delete)
	}
}

// RegisterResolveRoute mendaftarkan endpoint publik (tanpa auth) yang
// mengembalikan URL tujuan sebuah kunci shortlink sebagai JSON, dipanggil
// dari /api/v1/public/shortlinks/:key.
func RegisterResolveRoute(rg *gin.RouterGroup, h shortlink_handler.Handler) {
	rg.GET("/shortlinks/:key", h.Resolve)
}
