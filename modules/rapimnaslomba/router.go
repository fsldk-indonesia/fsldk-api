package rapimnaslomba

import (
	"fsldk-api/middlewares"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(
	api *gin.RouterGroup,
	pub *gin.RouterGroup,
	db *gorm.DB,
	mw *middlewares.Middleware,
) {
	h := newHandler(db)

	// Informasi lomba untuk pengunjung
	pub.GET(
		"/rapimnas/lomba",
		middlewares.RateLimit(30, 10),
		h.getPublic,
	)

	// Pengaturan lomba untuk CMS
	cms := api.Group("/rapimnas-lomba")
	cms.Use(mw.Auth(), mw.RequireVerified())

	cms.GET(
		"",
		mw.RequirePermission("rapimnas.lomba.view"),
		h.getSetting,
	)

	cms.PUT(
		"",
		mw.RequirePermission("rapimnas.lomba.manage"),
		h.updateSetting,
	)
}
