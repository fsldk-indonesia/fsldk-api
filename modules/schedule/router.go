// Package schedule wires up the schedule module's routing.
package schedule

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/schedule/schedule_handler"

	"github.com/gin-gonic/gin"
)

// RegisterPublicRoutes registers the schedule public endpoints.
func RegisterPublicRoutes(pub *gin.RouterGroup, h schedule_handler.Handler) {
	pub.GET("/schedules", h.PublicList)
}

// RegisterCMSRoutes registers the schedule CMS endpoints.
func RegisterCMSRoutes(rg *gin.RouterGroup, h schedule_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/schedules")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("", mw.RequirePermission(constants.PermScheduleView), h.CMSList)
		g.GET("/:id", mw.RequirePermission(constants.PermScheduleView), h.CMSGet)
		g.POST("", mw.RequirePermission(constants.PermScheduleCreate), h.Create)
		g.PUT("/:id", mw.RequirePermission(constants.PermScheduleUpdate), h.Update)
		g.PATCH("/:id/publish", mw.RequirePermission(constants.PermSchedulePublish), h.Publish)
		g.DELETE("/:id", mw.RequirePermission(constants.PermScheduleDelete), h.Delete)
	}
}
