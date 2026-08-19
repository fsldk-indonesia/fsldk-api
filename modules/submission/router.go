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
		g.POST("/:id/review", mw.RequirePermission(
			constants.PermSubmissionReviewLDK, constants.PermSubmissionReviewTier1,
			constants.PermSubmissionApproveTier1, constants.PermSubmissionReviewTier2,
		), h.Review)
		g.POST("/:id/establish-level", mw.RequirePermission(constants.PermSubmissionLevelEstablish), h.EstablishLevel)
		g.PUT("/:id/scores", mw.RequirePermission(constants.PermSubmissionReviewTier2), h.SaveFieldScores)
		g.POST("/:id/publish", mw.RequirePermission(constants.PermSubmissionPublish), h.Publish)
		g.POST("/:id/reopen", mw.RequirePermission(constants.PermSubmissionReopen), h.Reopen)
		g.POST("/:id/reassess", mw.RequirePermission(constants.PermSubmissionReassess), h.Reassess)
		g.POST("/:id/reassess-kader", mw.RequirePermission(constants.PermSubmissionCreate), h.ReassessKader)
	}

	k := rg.Group("/kaders")
	k.Use(mw.Auth(), mw.RequireVerified())
	{
		k.GET("", mw.RequirePermission(constants.PermSubmissionReviewLDK), h.ListKaders)
		k.GET("/:id/code", h.GetKaderCode)
		k.POST("/:id/deactivate", mw.RequirePermission(constants.PermKaderDeactivate), h.DeactivateKader)
		k.POST("/:id/reinstate", mw.RequirePermission(constants.PermKaderDeactivate), h.ReinstateKader)
	}
}
