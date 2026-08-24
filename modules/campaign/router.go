// Package campaign merangkai routing modul campaign (publik, milik-sendiri, CMS).
package campaign

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/campaign/campaign_handler"

	"github.com/gin-gonic/gin"
)

// RegisterPublicRoutes mendaftarkan endpoint publik campaign (tanpa autentikasi).
func RegisterPublicRoutes(pub *gin.RouterGroup, h campaign_handler.Handler) {
	pub.GET("/campaigns", h.PublicList)
	pub.GET("/campaigns/:slug", h.PublicDetail)
	pub.GET("/campaign-categories", h.Categories)
}

// RegisterMeRoutes mendaftarkan endpoint campaign milik pengguna sendiri (owner).
func RegisterMeRoutes(rg *gin.RouterGroup, h campaign_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/me/campaigns")
	g.Use(mw.Auth(), mw.RequireVerified(), mw.RequirePermission(constants.PermCampaignCreate))
	{
		g.GET("", h.MyList)
		g.POST("", h.Create)
		g.PUT("/:id", h.Update)
		g.PUT("/:id/beneficiary", h.UpdateBeneficiary)
		g.POST("/:id/submit", h.Submit)
	}
}

// RegisterCMSRoutes mendaftarkan endpoint CMS campaign (moderasi & monitoring).
func RegisterCMSRoutes(rg *gin.RouterGroup, h campaign_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/campaigns")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("", mw.RequirePermission(constants.PermCampaignView), h.CMSList)
		g.GET("/:id", mw.RequirePermission(constants.PermCampaignView), h.CMSGet)
		g.GET("/:id/review-history", mw.RequirePermission(constants.PermCampaignView), h.ReviewHistory)
		g.POST("/:id/review", mw.RequirePermission(constants.PermCampaignReview), h.Review)
		g.POST("/:id/publish", mw.RequirePermission(constants.PermCampaignPublish), h.Publish)
		g.POST("/:id/pause", mw.RequirePermission(constants.PermCampaignModerate), h.Pause)
		g.POST("/:id/resume", mw.RequirePermission(constants.PermCampaignModerate), h.Resume)
		g.POST("/:id/archive", mw.RequirePermission(constants.PermCampaignModerate), h.Archive)
	}
}
