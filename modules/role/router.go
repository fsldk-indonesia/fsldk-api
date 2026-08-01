// Package role merangkai routing modul role.
package role

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/role/role_handler"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes mendaftarkan endpoint modul role.
func RegisterRoutes(rg *gin.RouterGroup, h role_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/roles")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("", mw.RequirePermission(constants.PermRoleView), h.List)
		g.GET("/:id", mw.RequirePermission(constants.PermRoleView), h.Get)
		g.GET("/:id/users", mw.RequirePermission(constants.PermRoleView), h.Users)
		g.POST("", mw.RequirePermission(constants.PermRoleCreate), h.Create)
		g.PUT("/:id", mw.RequirePermission(constants.PermRoleUpdate), h.Update)
		g.PUT("/:id/permissions", mw.RequirePermission(constants.PermRoleUpdate), h.SetPermissions)
		g.DELETE("/:id", mw.RequirePermission(constants.PermRoleDelete), h.Delete)
	}
}
