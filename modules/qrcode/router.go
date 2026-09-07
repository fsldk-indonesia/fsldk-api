// Package qrcode merangkai routing modul QR Code (CRUD staff-only) dan QR Code
// request (alur permintaan publik + persetujuan admin di atasnya) — pola yang
// sama dengan modul shortlink, tapi artefaknya gambar QR yang meng-encode URL
// tujuan LANGSUNG (tanpa kunci / redirect / pelacakan pindaian).
package qrcode

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/qrcode/qrcode_handler"
	"fsldk-api/modules/qrcode/qrcoderequest_handler"

	"github.com/gin-gonic/gin"
)

// RegisterCMSRoutes mendaftarkan endpoint manajemen QR Code (terproteksi
// auth + verifikasi + permission), di bawah grup /api/v1.
func RegisterCMSRoutes(rg *gin.RouterGroup, h qrcode_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/qrcodes")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("", mw.RequirePermission(constants.PermQRCodeView), h.List)
		g.GET("/:id", mw.RequirePermission(constants.PermQRCodeView), h.Get)
		g.POST("", mw.RequirePermission(constants.PermQRCodeCreate), h.Create)
		g.PUT("/:id", mw.RequirePermission(constants.PermQRCodeUpdate), h.Update)
		g.DELETE("/:id", mw.RequirePermission(constants.PermQRCodeDelete), h.Delete)
	}
}

// RegisterImageRoute mendaftarkan endpoint publik (tanpa auth) yang
// mengembalikan gambar PNG QR untuk sebuah baris, dipanggil dari
// /api/v1/public/qrcodes/:id/image.
func RegisterImageRoute(rg *gin.RouterGroup, h qrcode_handler.Handler) {
	rg.GET("/qrcodes/:id/image", h.Image)
}

// RegisterRequestPublicRoutes mendaftarkan endpoint publik modul QR Code
// request: Submit (rate-limited seperti shortlink-requests) dan info PIC.
func RegisterRequestPublicRoutes(pub *gin.RouterGroup, h qrcoderequest_handler.Handler) {
	pub.POST("/qrcode-requests", middlewares.RateLimit(3, 3), h.Submit) // 3/menit, burst 3
	pub.GET("/qrcode-requests/pic", h.PublicPIC)
}

// RegisterRequestCMSRoutes mendaftarkan endpoint antrian moderasi QR Code
// request.
func RegisterRequestCMSRoutes(rg *gin.RouterGroup, h qrcoderequest_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/qrcode-requests")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("", mw.RequirePermission(constants.PermQRCodeView), h.CMSList)
		g.GET("/:id", mw.RequirePermission(constants.PermQRCodeView), h.CMSGet)
		g.POST("/:id/approve", mw.RequirePermission(constants.PermQRCodeApprove), h.Approve)
		g.POST("/:id/reject", mw.RequirePermission(constants.PermQRCodeApprove), h.Reject)
	}
}
