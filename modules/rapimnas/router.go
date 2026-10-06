// Package rapimnas merangkai routing modul rapimnas (CMS Super-Admin-only
// "Rapimnas Setup" + satu endpoint publik tanpa auth untuk 6 halaman
// /rapimnas).
package rapimnas

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/rapimnas/rapimnas_handler"

	"github.com/gin-gonic/gin"
)

// RegisterCMSRoutes mendaftarkan endpoint manajemen Rapimnas Setup.
func RegisterCMSRoutes(rg *gin.RouterGroup, h rapimnas_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/rapimnas-setup")
	g.Use(mw.Auth(), mw.RequireVerified())
	g.GET("", mw.RequirePermission(constants.PermRapimnasView), h.Get)
	g.PUT("", mw.RequirePermission(constants.PermRapimnasUpdate), h.Update)
}

// RegisterPublicRoutes mendaftarkan GET /public/rapimnas — dipanggil 6
// halaman publik /rapimnas, rate limit ringan sekadar jaga dari flooding
// trivial (pola sama dengan welcomepopup.RegisterPublicRoutes).
func RegisterPublicRoutes(pub *gin.RouterGroup, h rapimnas_handler.Handler) {
	pub.GET("/rapimnas", middlewares.RateLimit(30, 10), h.GetPublic)
}
