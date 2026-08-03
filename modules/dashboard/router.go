// Package dashboard merangkai routing modul dashboard.
package dashboard

import (
	"fsldk-api/middlewares"
	"fsldk-api/modules/dashboard/dashboard_handler"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes mendaftarkan endpoint dashboard.
func RegisterRoutes(rg *gin.RouterGroup, h dashboard_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/dashboard")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("/summary", h.Summary)
	}
}
