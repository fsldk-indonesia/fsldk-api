// Package submission merangkai routing modul submission.
package submission

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/submission/submission_handler"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes mendaftarkan endpoint modul submission. Scope kepemilikan
// data (siapa boleh mengubah/melihat submission mana) divalidasi di lapisan
// service (submission_service), bukan lewat RequireOrganizationScope —
// submission ORGANIZATION-subject dikunci ke organisasi pemanggil sendiri,
// submission KADER-subject memilih organisasi tujuan bebas (LDK manapun),
// sehingga tidak bisa diwakili satu aturan scope path/query param generik.
func RegisterRoutes(rg *gin.RouterGroup, h submission_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/submissions")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.POST("", mw.RequirePermission(constants.PermSubmissionCreate), h.Create)
		g.PUT("/:id/answers", mw.RequirePermission(constants.PermSubmissionUpdate), h.SaveAnswers)
		g.POST("/:id/submit", mw.RequirePermission(constants.PermSubmissionCreate), h.Submit)
		g.POST("/:id/cancel", mw.RequirePermission(constants.PermSubmissionCancel), h.Cancel)
		g.GET("", mw.RequirePermission(constants.PermSubmissionView), h.List)
		g.GET("/:id", mw.RequirePermission(constants.PermSubmissionView), h.Get)
	}
}
