// Package donation merangkai routing modul donation (publik, milik-sendiri, CMS).
package donation

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/donation/donation_handler"

	"github.com/gin-gonic/gin"
)

// RegisterPublicRoutes mendaftarkan endpoint publik donation. OptionalAuth
// dipasang pada submit donasi agar donatur yang sedang login tertaut
// (donorUserID), tanpa mewajibkan login (donasi tamu tetap didukung).
func RegisterPublicRoutes(pub *gin.RouterGroup, h donation_handler.Handler, mw *middlewares.Middleware) {
	pub.POST("/campaigns/:slug/donate", mw.OptionalAuth(), middlewares.RateLimit(30, 5), h.Create)
	pub.GET("/campaigns/:slug/donations", h.PublicRecentDonations)
	pub.GET("/donations/:publicRef", h.Detail)
	pub.GET("/donations/:publicRef/status", middlewares.RateLimit(60, 10), h.Status)
}

// RegisterCallbackRoutes mendaftarkan webhook payment provider — bukan di
// bawah /public (beda audiens: server-to-server, bukan klien publik),
// mengikuti kontrak API §7.3. Diamankan via IP allowlist opsional +
// signature (donation_service.ProcessCallback), bukan JWT.
func RegisterCallbackRoutes(api *gin.RouterGroup, h donation_handler.Handler, mw *middlewares.Middleware) {
	api.POST("/payments/callback",
		middlewares.IPAllowlist(mw.Cfg.AllowedBisatopupIPsCrowdfunding()),
		middlewares.RateLimit(120, 20),
		h.Callback)
}

// RegisterMeRoutes mendaftarkan endpoint riwayat donasi milik pengguna sendiri.
func RegisterMeRoutes(rg *gin.RouterGroup, h donation_handler.Handler, mw *middlewares.Middleware) {
	rg.GET("/me/donations", mw.Auth(), mw.RequireVerified(), h.MyList)
}

// RegisterCMSRoutes mendaftarkan endpoint CMS monitoring donasi.
func RegisterCMSRoutes(rg *gin.RouterGroup, h donation_handler.Handler, mw *middlewares.Middleware) {
	rg.GET("/donations", mw.Auth(), mw.RequireVerified(), mw.RequirePermission(constants.PermDonationView), h.CMSList)
}
