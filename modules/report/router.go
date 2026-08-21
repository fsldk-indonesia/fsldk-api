// Package report merangkai routing modul report.
package report

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/report/report_handler"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes mendaftarkan endpoint modul report.
func RegisterRoutes(rg *gin.RouterGroup, h report_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/reports")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("/submissions/export", mw.RequirePermission(
			constants.PermReportRegionExport, constants.PermReportNationalExport,
		), h.ExportSubmissions)
	}
}
