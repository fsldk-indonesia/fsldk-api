// Package wallet merangkai routing modul wallet (CMS).
package wallet

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/wallet/wallet_handler"

	"github.com/gin-gonic/gin"
)

// RegisterCMSRoutes mendaftarkan endpoint balance/ledger campaign untuk
// admin CMS — TIDAK ada lagi endpoint milik-sendiri (revisi 2026-09-01,
// campaign tidak lagi punya kepemilikan; balance/ledger self-service
// digantikan Laporan Kantong Amal, item 6 revision-prompt-2.md).
func RegisterCMSRoutes(rg *gin.RouterGroup, h wallet_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/campaigns/:id")
	g.Use(mw.Auth(), mw.RequireVerified(), mw.RequirePermission(constants.PermWalletView))
	{
		g.GET("/balance", h.CMSBalance)
		g.GET("/ledger", h.CMSLedger)
	}
}
