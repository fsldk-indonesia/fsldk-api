// Package withdrawal merangkai routing modul withdrawal (milik-sendiri, CMS, callback).
package withdrawal

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/withdrawal/withdrawal_handler"

	"github.com/gin-gonic/gin"
)

// RegisterMeRoutes mendaftarkan endpoint withdrawal milik campaign owner sendiri.
func RegisterMeRoutes(rg *gin.RouterGroup, h withdrawal_handler.Handler, mw *middlewares.Middleware) {
	rg.POST("/me/campaigns/:id/withdrawals", mw.Auth(), mw.RequireVerified(), mw.RequirePermission(constants.PermWithdrawalRequest), h.Request)
	rg.GET("/me/withdrawals", mw.Auth(), mw.RequireVerified(), h.MyList)
	rg.POST("/me/withdrawals/:id/cancel", mw.Auth(), mw.RequireVerified(), h.Cancel)

	rg.GET("/transfer/banks", mw.Auth(), mw.RequireVerified(), h.ListBanks)
	rg.POST("/transfer/inquiry", mw.Auth(), mw.RequireVerified(), h.Inquiry)
}

// RegisterCMSRoutes mendaftarkan endpoint CMS approval/proses withdrawal.
func RegisterCMSRoutes(rg *gin.RouterGroup, h withdrawal_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/withdrawals")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("", mw.RequirePermission(constants.PermWithdrawalApprove), h.CMSList)
		g.POST("/:id/approve", mw.RequirePermission(constants.PermWithdrawalApprove), h.Approve)
		g.POST("/:id/reject", mw.RequirePermission(constants.PermWithdrawalReject), h.Reject)
		g.POST("/:id/process", mw.RequirePermission(constants.PermWithdrawalProcess), h.Process)
	}
}

// RegisterCallbackRoutes mendaftarkan webhook disbursement provider — bukan
// di bawah /public (server-to-server), diamankan URL path secret (bukan
// JWT/signature, lihat withdrawal_service.ProcessCallback).
func RegisterCallbackRoutes(api *gin.RouterGroup, h withdrawal_handler.Handler, mw *middlewares.Middleware) {
	api.POST("/withdrawals/callback/:secret",
		middlewares.IPAllowlist(mw.Cfg.AllowedBisatopupIPsCrowdfunding()),
		middlewares.RateLimit(60, 10),
		h.Callback)
}
