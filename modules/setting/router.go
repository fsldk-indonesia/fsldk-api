// Package setting merangkai routing modul setting (App Settings — konfigurasi
// runtime platform generik, Superadmin-only). Tidak ada sisi publik.
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
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("", mw.RequirePermission(constants.PermSettingView), h.List)
		g.PUT("/:id", mw.RequirePermission(constants.PermSettingUpdate), h.Update)
	}
}
