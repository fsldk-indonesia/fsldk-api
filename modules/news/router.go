package news

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"

	"github.com/gin-gonic/gin"
)

// RegisterPublicRoutes mendaftarkan endpoint publik berita (tanpa autentikasi).
func RegisterPublicRoutes(pub *gin.RouterGroup, h *Handler) {
	pub.GET("/news", h.PublicList)
	pub.GET("/news-featured", h.PublicFeatured)
	pub.GET("/news-categories", h.Categories)
	pub.GET("/news/:slug", h.PublicDetail)
}

// RegisterCMSRoutes mendaftarkan endpoint CMS berita (terproteksi).
func RegisterCMSRoutes(rg *gin.RouterGroup, h *Handler, mw *middlewares.Middleware) {
	g := rg.Group("/news")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("", mw.RequirePermission(constants.PermNewsView), h.CMSList)
		g.GET("/:id", mw.RequirePermission(constants.PermNewsView), h.CMSGet)
		g.POST("", mw.RequirePermission(constants.PermNewsCreate), h.Create)
		g.PUT("/:id", mw.RequirePermission(constants.PermNewsUpdate), h.Update)
		g.PATCH("/:id/publish", mw.RequirePermission(constants.PermNewsPublish), h.Publish)
		g.PATCH("/:id/featured", mw.RequirePermission(constants.PermNewsUpdate), h.Featured)
		g.DELETE("/:id", mw.RequirePermission(constants.PermNewsDelete), h.Delete)
	}
}
