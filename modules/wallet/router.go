// Package wallet merangkai routing modul wallet (milik-sendiri, CMS).
package wallet

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/wallet/wallet_handler"

	"github.com/gin-gonic/gin"
)

// RegisterMeRoutes mendaftarkan endpoint balance/ledger milik campaign
// owner sendiri. Kepemilikan campaign divalidasi di wallet_service (404
// bila bukan pemilik), permission di sini hanya gerbang umum "boleh
// mengelola campaign sendiri" — pola sama seperti modules/campaign.
func RegisterMeRoutes(rg *gin.RouterGroup, h wallet_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/me/campaigns/:id")
	g.Use(mw.Auth(), mw.RequireVerified(), mw.RequirePermission(constants.PermCampaignCreate))
	{
		g.GET("/balance", h.MyBalance)
		g.GET("/ledger", h.MyLedger)
	}
}

// RegisterCMSRoutes mendaftarkan endpoint balance untuk admin.
func RegisterCMSRoutes(rg *gin.RouterGroup, h wallet_handler.Handler, mw *middlewares.Middleware) {
	rg.GET("/campaigns/:id/balance", mw.Auth(), mw.RequireVerified(), mw.RequirePermission(constants.PermWalletView), h.CMSBalance)
}
