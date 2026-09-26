// Package welcomepopup merangkai routing modul welcomepopup (popup Beranda
// dinamis — CMS Super-Admin-only + satu endpoint publik tanpa auth).
package welcomepopup

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/welcomepopup/welcomepopup_handler"

	"github.com/gin-gonic/gin"
)

// RegisterCMSRoutes mendaftarkan endpoint manajemen Welcome Popup.
func RegisterCMSRoutes(rg *gin.RouterGroup, h welcomepopup_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/welcome-popup")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("", mw.RequirePermission(constants.PermWelcomePopupView), h.Get)
		g.PUT("", mw.RequirePermission(constants.PermWelcomePopupUpdate), h.Update)
	}
}

// RegisterPublicRoutes mendaftarkan GET /public/welcome-popup — dipanggil
// halaman Beranda publik, rate limit ringan sekadar jaga dari flooding
// trivial (pola sama dengan zakat.RegisterPublicRoutes).
func RegisterPublicRoutes(pub *gin.RouterGroup, h welcomepopup_handler.Handler) {
	pub.GET("/welcome-popup", middlewares.RateLimit(30, 10), h.GetPublic)
}
