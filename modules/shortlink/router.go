// Package shortlink merangkai routing modul shortlink (CRUD staff-only) dan
// shortlink request (alur permintaan publik + persetujuan admin di atasnya).
package shortlink

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/shortlink/shortlink_handler"
	"fsldk-api/modules/shortlink/shortlinkrequest_handler"

	"github.com/gin-gonic/gin"
)

// RegisterCMSRoutes mendaftarkan endpoint manajemen shortlink (terproteksi
// auth + verifikasi + permission), di bawah grup /api/v1.
func RegisterCMSRoutes(rg *gin.RouterGroup, h shortlink_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/shortlinks")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("", mw.RequirePermission(constants.PermShortlinkView), h.List)
		g.GET("/:id", mw.RequirePermission(constants.PermShortlinkView), h.Get)
		g.POST("", mw.RequirePermission(constants.PermShortlinkCreate), h.Create)
		g.PUT("/:id", mw.RequirePermission(constants.PermShortlinkUpdate), h.Update)
		g.DELETE("/:id", mw.RequirePermission(constants.PermShortlinkDelete), h.Delete)
	}
}

// RegisterResolveRoute mendaftarkan endpoint publik (tanpa auth) yang
// mengembalikan URL tujuan sebuah kunci shortlink sebagai JSON, dipanggil
// dari /api/v1/public/shortlinks/:key.
func RegisterResolveRoute(rg *gin.RouterGroup, h shortlink_handler.Handler) {
	rg.GET("/shortlinks/:key", h.Resolve)
}

// RegisterRequestPublicRoutes mendaftarkan endpoint publik modul shortlink
// request: Submit (rate-limited, seperti register/login modules/auth — ini
// konsumen RateLimit baru pertama di luar auth) dan webhook Kirimdev
// (signature diverifikasi di handler, bukan middleware).
func RegisterRequestPublicRoutes(pub *gin.RouterGroup, h shortlinkrequest_handler.Handler) {
	pub.POST("/shortlink-requests", middlewares.RateLimit(3, 3), h.Submit) // 3/menit, burst 3
	pub.GET("/shortlink-requests/pic", h.PublicPIC)
	pub.POST("/webhooks/kirimdev", h.KirimdevWebhook)
}

// RegisterRequestCMSRoutes mendaftarkan endpoint antrian moderasi shortlink
// request. Path /shortlink-requests sengaja statis (base path terpisah dari
// /shortlinks) supaya permission gating shortlink.approve vs shortlink.* lain
// tetap jelas dipisah per resource.
func RegisterRequestCMSRoutes(rg *gin.RouterGroup, h shortlinkrequest_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/shortlink-requests")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("", mw.RequirePermission(constants.PermShortlinkView), h.CMSList)
		g.GET("/:id", mw.RequirePermission(constants.PermShortlinkView), h.CMSGet)
		g.POST("/:id/approve", mw.RequirePermission(constants.PermShortlinkApprove), h.Approve)
		g.POST("/:id/reject", mw.RequirePermission(constants.PermShortlinkApprove), h.Reject)
	}
}
