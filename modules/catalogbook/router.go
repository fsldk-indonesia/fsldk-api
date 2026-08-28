// Package catalogbook wires up the catalogbook module's routing.
package catalogbook

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/catalogbook/catalogbook_handler"

	"github.com/gin-gonic/gin"
)

// RegisterPublicRoutes registers the catalogbook public endpoints.
func RegisterPublicRoutes(pub *gin.RouterGroup, h catalogbook_handler.Handler) {
	pub.GET("/catalog-books", h.PublicList)
	pub.GET("/catalog-books/:slug", h.PublicDetail)
	// Rate-limited: the Laravel reference never throttled this endpoint at
	// all — a spam gap closed here (same pattern as shortlink-requests).
	pub.POST("/catalog-books/:id/like", middlewares.RateLimit(20, 5), h.Like)
	pub.GET("/catalog-book-categories", h.Categories)
	pub.GET("/catalog-book-languages", h.Languages)
	pub.GET("/catalog-book-author-types", h.AuthorTypes)
	pub.GET("/catalog-book-availability-types", h.AvailabilityTypes)
}

// RegisterCMSRoutes registers the catalogbook CMS endpoints.
func RegisterCMSRoutes(rg *gin.RouterGroup, h catalogbook_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/catalog-books")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("", mw.RequirePermission(constants.PermCatalogBookView), h.CMSList)
		g.GET("/:id", mw.RequirePermission(constants.PermCatalogBookView), h.CMSGet)
		g.POST("", mw.RequirePermission(constants.PermCatalogBookCreate), h.Create)
		g.PUT("/:id", mw.RequirePermission(constants.PermCatalogBookUpdate), h.Update)
		g.PATCH("/:id/publish", mw.RequirePermission(constants.PermCatalogBookPublish), h.Publish)
		g.DELETE("/:id", mw.RequirePermission(constants.PermCatalogBookDelete), h.Delete)
	}
}
