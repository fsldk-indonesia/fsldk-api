// Package permission merangkai routing modul permission & menu.
package permission

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/permission/permission_handler"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes mendaftarkan endpoint permission & menu.
func RegisterRoutes(rg *gin.RouterGroup, h permission_handler.Handler, mw *middlewares.Middleware) {
	// Daftar seluruh permission (untuk halaman manajemen role).
	rg.GET("/permissions", mw.Auth(), mw.RequireVerified(), mw.RequirePermission(constants.PermRoleView), h.ListAll)

	// Menu sidebar CMS: cukup terautentikasi + terverifikasi (tanpa permission khusus).
	rg.GET("/me/menus", mw.Auth(), mw.RequireVerified(), h.Menu)
}
