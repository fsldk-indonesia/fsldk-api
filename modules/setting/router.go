// Package setting merangkai routing modul setting (App Settings — konfigurasi
// runtime platform generik, Superadmin-only). Dua pengecualian publik:
// GET /public/settings/contact-email dan GET /public/settings/contact-whatsapp
// (lihat setting_model.GroupKontak), dikonsumsi langsung oleh beranda,
// floating button WhatsApp, & halaman /tentang/kontak — nilai setting lain
// TIDAK ikut terekspos lewat endpoint ini.
package setting

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/setting/setting_handler"

	"github.com/gin-gonic/gin"
)

// RegisterCMSRoutes mendaftarkan endpoint manajemen App Settings.
func RegisterCMSRoutes(rg *gin.RouterGroup, h setting_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/settings")
	g.Use(mw.Auth())
	{
		g.GET("", mw.RequirePermission(constants.PermSettingView), h.List)
		g.PUT("/:id", mw.RequirePermission(constants.PermSettingUpdate), h.Update)
	}
}

// RegisterPublicRoutes mendaftarkan endpoint publik kontak (email & WhatsApp).
func RegisterPublicRoutes(rg *gin.RouterGroup, h setting_handler.Handler) {
	rg.GET("/settings/contact-email", h.PublicContactEmail)
	rg.GET("/settings/contact-whatsapp", h.PublicContactWhatsapp)
}
