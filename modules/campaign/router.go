// Package campaign merangkai routing modul campaign (publik & CMS).
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

// RegisterCMSRoutes mendaftarkan endpoint CMS campaign — CRUD murni
// permission gate per aksi, TIDAK ada lagi kepemilikan/self-service
// (revisi 2026-09-01 — siapapun dengan permission terkait boleh mengelola
// campaign manapun, mengikuti model celengan syahid ldksyahid-app).
func RegisterCMSRoutes(rg *gin.RouterGroup, h campaign_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/campaigns")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("", mw.RequirePermission(constants.PermCampaignView), h.CMSList)
		g.GET("/lite", mw.RequirePermission(constants.PermCampaignView), h.CMSListLite)
		g.GET("/:id", mw.RequirePermission(constants.PermCampaignView), h.CMSGet)
		g.POST("", mw.RequirePermission(constants.PermCampaignCreate), h.Create)
		g.PUT("/:id", mw.RequirePermission(constants.PermCampaignUpdate), h.Update)
		g.DELETE("/:id", mw.RequirePermission(constants.PermCampaignDelete), h.Delete)
		g.POST("/:id/publish", mw.RequirePermission(constants.PermCampaignPublish), h.Publish)
		g.POST("/:id/pause", mw.RequirePermission(constants.PermCampaignModerate), h.Pause)
		g.POST("/:id/resume", mw.RequirePermission(constants.PermCampaignModerate), h.Resume)
		g.POST("/:id/archive", mw.RequirePermission(constants.PermCampaignModerate), h.Archive)
	}
}
