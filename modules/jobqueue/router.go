// Package jobqueue merangkai routing modul job queue (dashboard monitoring
// antrian pengiriman asinkron, §1b techspec). CMS-only, Super Admin only —
// tidak ada sisi publik, sama posture-nya dengan modules/setting.
package jobqueue

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/jobqueue/jobqueue_handler"

	"github.com/gin-gonic/gin"
)

// RegisterCMSRoutes mendaftarkan endpoint dashboard job queue.
func RegisterCMSRoutes(rg *gin.RouterGroup, h jobqueue_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/job-queue")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("", mw.RequirePermission(constants.PermJobQueueView), h.CMSList)
		g.GET("/stats", mw.RequirePermission(constants.PermJobQueueView), h.CMSStats)
		g.GET("/:id", mw.RequirePermission(constants.PermJobQueueView), h.CMSGet)
		g.POST("/:id/retry", mw.RequirePermission(constants.PermJobQueueRetry), h.Retry)
		g.DELETE("/:id", mw.RequirePermission(constants.PermJobQueueDelete), h.Delete)
	}
}
