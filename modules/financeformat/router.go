// Package financeformat wires up the financeformat module's routing.
package financeformat

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/financeformat/financeformat_handler"

	"github.com/gin-gonic/gin"
)

// RegisterPublicRoutes registers the financeformat public endpoints.
func RegisterPublicRoutes(pub *gin.RouterGroup, h financeformat_handler.Handler) {
	// One call returns { formatTypes, formats, cpName, cpPhone } — the
	// frontend groups formats by formatTypeID client-side.
	pub.GET("/finance-formats", h.PublicList)
	// Download by id. The trailing :name segment is decorative (kebab-case of
	// the file name) so a copied link is readable; the served filename is set
	// from the DB, not this segment.
	pub.GET("/finance-formats/:id/download", h.Download)
	pub.GET("/finance-formats/:id/download/:name", h.Download)
}

// RegisterCMSRoutes registers the financeformat CMS endpoints.
func RegisterCMSRoutes(rg *gin.RouterGroup, h financeformat_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/finance-formats")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("", mw.RequirePermission(constants.PermFinanceFormatView), h.CMSList)
		g.GET("/types", mw.RequirePermission(constants.PermFinanceFormatView), h.FormatTypes)
		g.GET("/:id", mw.RequirePermission(constants.PermFinanceFormatView), h.CMSGet)
		g.POST("", mw.RequirePermission(constants.PermFinanceFormatCreate), h.Create)
		g.PUT("/:id", mw.RequirePermission(constants.PermFinanceFormatUpdate), h.Update)
		g.PATCH("/:id/publish", mw.RequirePermission(constants.PermFinanceFormatPublish), h.Publish)
		g.DELETE("/:id", mw.RequirePermission(constants.PermFinanceFormatDelete), h.Delete)
	}
}
