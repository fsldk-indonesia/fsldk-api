// Package content merangkai routing modul content.
package content

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/content/content_handler"

	"github.com/gin-gonic/gin"
)

// RegisterPublicRoutes mendaftarkan endpoint publik konten & struktur organisasi.
func RegisterPublicRoutes(pub *gin.RouterGroup, h content_handler.Handler) {
	pub.GET("/contents", h.PublicList)
	pub.GET("/contents/:key", h.PublicByKey)
	pub.GET("/profile", h.PublicProfile)
	pub.GET("/organization-structure", h.PublicOrg)
}

// RegisterCMSRoutes mendaftarkan endpoint CMS konten & struktur organisasi.
func RegisterCMSRoutes(rg *gin.RouterGroup, h content_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("/contents", mw.RequirePermission(constants.PermContentView), h.CMSList)
		g.PUT("/contents/:key", mw.RequirePermission(constants.PermContentUpdate), h.Update)
		g.GET("/organization-structure", mw.RequirePermission(constants.PermContentView), h.OrgList)
		g.POST("/organization-structure", mw.RequirePermission(constants.PermContentUpdate), h.OrgCreate)
		g.PUT("/organization-structure/:id", mw.RequirePermission(constants.PermContentUpdate), h.OrgUpdate)
		g.DELETE("/organization-structure/:id", mw.RequirePermission(constants.PermContentUpdate), h.OrgDelete)
	}
}
