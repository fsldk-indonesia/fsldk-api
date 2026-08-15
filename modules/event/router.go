// Package event wires routing for the event module.
package event

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/event/event_handler"

	"github.com/gin-gonic/gin"
)

// RegisterPublicRoutes registers unauthenticated event endpoints.
func RegisterPublicRoutes(pub *gin.RouterGroup, h event_handler.Handler) {
	pub.GET("/events", h.ListPublic)
	pub.GET("/events/:slug", h.ShowPublic)
}

// RegisterCMSRoutes registers protected CMS event endpoints.
func RegisterCMSRoutes(rg *gin.RouterGroup, h event_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/events")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("", mw.RequirePermission(constants.PermEventView), h.ListCMS)
		g.GET("/:id", mw.RequirePermission(constants.PermEventView), h.ShowCMS)
		g.POST("", mw.RequirePermission(constants.PermEventCreate), h.Create)
		g.PUT("/:id", mw.RequirePermission(constants.PermEventUpdate), h.Update)
		g.DELETE("/:id", mw.RequirePermission(constants.PermEventDelete), h.Delete)
	}
}
