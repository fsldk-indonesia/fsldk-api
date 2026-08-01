// Package article merangkai routing modul article.
package article

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/article/article_handler"

	"github.com/gin-gonic/gin"
)

// RegisterPublicRoutes mendaftarkan endpoint publik artikel.
func RegisterPublicRoutes(pub *gin.RouterGroup, h article_handler.Handler) {
	pub.GET("/articles", h.PublicList)
	pub.GET("/article-categories", h.Categories)
	pub.GET("/articles/:slug", h.PublicDetail)
}

// RegisterCMSRoutes mendaftarkan endpoint CMS artikel.
func RegisterCMSRoutes(rg *gin.RouterGroup, h article_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/articles")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("", mw.RequirePermission(constants.PermArticleView), h.CMSList)
		g.GET("/:id", mw.RequirePermission(constants.PermArticleView), h.CMSGet)
		g.POST("", mw.RequirePermission(constants.PermArticleCreate), h.Create)
		g.PUT("/:id", mw.RequirePermission(constants.PermArticleUpdate), h.Update)
		g.PATCH("/:id/publish", mw.RequirePermission(constants.PermArticlePublish), h.Publish)
		g.DELETE("/:id", mw.RequirePermission(constants.PermArticleDelete), h.Delete)
	}
}
