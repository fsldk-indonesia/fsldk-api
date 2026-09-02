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

// RegisterCMSRoutes mendaftarkan endpoint CMS monitoring donasi + CRUD
// donasi manual/offline (item 1 revision-prompt-2.md — donasi yang tidak
// lewat Amdigipay/Bisatopup). AdminUpdate/AdminDelete menolak donasi
// gateway="bisatopup" di service layer (lihat donation_service_impl.go).
func RegisterCMSRoutes(rg *gin.RouterGroup, h donation_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/donations")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("", mw.RequirePermission(constants.PermDonationView), h.CMSList)
		g.GET("/:id", mw.RequirePermission(constants.PermDonationView), h.CMSGet)
		g.POST("", mw.RequirePermission(constants.PermDonationCreate), h.AdminCreate)
		g.PUT("/:id", mw.RequirePermission(constants.PermDonationUpdate), h.AdminUpdate)
		g.DELETE("/:id", mw.RequirePermission(constants.PermDonationDelete), h.AdminDelete)
	}
}
