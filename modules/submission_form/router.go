// Package submission_form merangkai routing modul submission_form (form builder).
package submission_form

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/submission_form/submission_form_handler"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes mendaftarkan endpoint form builder. Seluruh endpoint hanya
// untuk Super Admin (submission_form.view untuk baca, submission_form.manage
// untuk seluruh operasi tulis).
func RegisterRoutes(rg *gin.RouterGroup, h submission_form_handler.Handler, mw *middlewares.Middleware) {
	view := mw.RequirePermission(constants.PermSubmissionFormView)
	manage := mw.RequirePermission(constants.PermSubmissionFormManage)

	g := rg.Group("/submission-forms")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		// Struktur form PUBLISHED (bukan jawaban) bukan data sensitif — siapa
		// pun yang login & terverifikasi boleh membacanya untuk merender form
		// pengisian, tanpa perlu permission admin form builder.
		g.GET("/by-code/:formCode/published", h.GetPublishedByFormCode)

		g.GET("", view, h.ListForms)
		g.POST("", manage, h.CreateForm)
		g.GET("/:formID", view, h.GetForm)
		g.POST("/:formID/versions", manage, h.CreateVersion)

		g.GET("/versions/:versionID", view, h.GetVersion)
		g.POST("/versions/:versionID/publish", manage, h.PublishVersion)
		g.POST("/versions/:versionID/sections", manage, h.CreateSection)

		g.PUT("/sections/:sectionID", manage, h.UpdateSection)
		g.DELETE("/sections/:sectionID", manage, h.DeleteSection)
		g.POST("/sections/:sectionID/fields", manage, h.CreateField)

		g.PUT("/fields/:fieldID", manage, h.UpdateField)
		g.DELETE("/fields/:fieldID", manage, h.DeleteField)
		g.POST("/fields/:fieldID/options", manage, h.CreateOption)

		g.PUT("/options/:optionID", manage, h.UpdateOption)
		g.DELETE("/options/:optionID", manage, h.DeleteOption)
	}
}
