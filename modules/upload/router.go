// Package upload merangkai routing modul upload.
package upload

import (
	"fsldk-api/middlewares"
	"fsldk-api/modules/upload/upload_handler"

	"github.com/gin-gonic/gin"
)

// RegisterCMSRoutes mendaftarkan endpoint unggah berkas gambar (terproteksi
// auth + verifikasi) yang dipakai bersama oleh form Artikel & Berita CMS.
func RegisterCMSRoutes(rg *gin.RouterGroup, h upload_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/uploads")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.POST("/image", h.UploadImage)
	}
}
