// Package dynamicform wires up the dynamicform module's routing.
package dynamicform

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/dynamicform/dynamicform_handler"

	"github.com/gin-gonic/gin"
)

// RegisterPublicRoutes registers the public (fill) endpoints under
// /api/v1/public/dynamic-forms. OptionalAuth sets the caller identity when a
// valid token is present without ever rejecting a guest.
func RegisterPublicRoutes(pub *gin.RouterGroup, h dynamicform_handler.Handler, mw *middlewares.Middleware) {
	g := pub.Group("/dynamic-forms")
	g.Use(mw.OptionalAuth())
	{
		g.GET("/:slug", h.PublicGet)
		g.POST("/:slug/submit", middlewares.RateLimit(10, 5), h.PublicSubmit)

		// Draft autosave — login required (server-side draft is per account).
		// Each route has its own rate limiter so draft-save traffic never eats
		// the submit route's budget.
		d := g.Group("/:slug/draft")
		d.Use(mw.Auth())
		{
			d.POST("", middlewares.RateLimit(40, 10), h.SaveDraft)
			d.POST("/file/:fieldID", middlewares.RateLimit(20, 5), h.StageDraftFile)
			d.DELETE("/file/:fieldID", middlewares.RateLimit(30, 5), h.RemoveDraftFile)
		}
	}
}

// RegisterCMSRoutes registers the CMS endpoints under /api/v1/dynamic-forms.
func RegisterCMSRoutes(rg *gin.RouterGroup, h dynamicform_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/dynamic-forms")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("", mw.RequirePermission(constants.PermDynamicFormView), h.CMSList)
		g.POST("", mw.RequirePermission(constants.PermDynamicFormCreate), h.Create)
		g.GET("/:id", mw.RequirePermission(constants.PermDynamicFormView), h.CMSGet)
		g.PUT("/:id", mw.RequirePermission(constants.PermDynamicFormUpdate), h.Update)
		g.PATCH("/:id/status", mw.RequirePermission(constants.PermDynamicFormPublish), h.SetStatus)
		g.DELETE("/:id", mw.RequirePermission(constants.PermDynamicFormDelete), h.Delete)
		g.POST("/bulk-delete", mw.RequirePermission(constants.PermDynamicFormDelete), h.BulkDelete)

		g.POST("/:id/fields", mw.RequirePermission(constants.PermDynamicFormUpdate), h.AddField)
		g.PUT("/:id/fields/:fieldID", mw.RequirePermission(constants.PermDynamicFormUpdate), h.UpdateField)
		g.DELETE("/:id/fields/:fieldID", mw.RequirePermission(constants.PermDynamicFormUpdate), h.RemoveField)
		g.POST("/:id/fields/reorder", mw.RequirePermission(constants.PermDynamicFormUpdate), h.ReorderFields)

		g.GET("/:id/submissions", mw.RequirePermission(constants.PermDynamicFormView), h.ListSubmissions)
		g.GET("/:id/submissions/:subID", mw.RequirePermission(constants.PermDynamicFormView), h.GetSubmission)
		g.PUT("/:id/submissions/:subID", mw.RequirePermission(constants.PermDynamicFormUpdate), h.UpdateSubmission)
		g.DELETE("/:id/submissions/:subID", mw.RequirePermission(constants.PermDynamicFormDelete), h.DeleteSubmission)
		g.GET("/:id/responses.csv", mw.RequirePermission(constants.PermDynamicFormView), h.ExportCSV)
		g.DELETE("/:id/submissions", mw.RequirePermission(constants.PermDynamicFormDelete), h.DeleteResponses)
		g.GET("/:id/analytics", mw.RequirePermission(constants.PermDynamicFormView), h.Analytics)

		g.POST("/:id/gsheet/connect", mw.RequirePermission(constants.PermDynamicFormUpdate), h.GSheetConnect)
		g.POST("/:id/gsheet/resync", mw.RequirePermission(constants.PermDynamicFormUpdate), h.GSheetResync)
		g.DELETE("/:id/gsheet", mw.RequirePermission(constants.PermDynamicFormUpdate), h.GSheetDisconnect)
	}
}
