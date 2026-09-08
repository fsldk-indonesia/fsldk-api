// Package goods merangkai routing modul goods.
package goods

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/goods/goods_handler"

	"github.com/gin-gonic/gin"
)

// RegisterPublicRoutes mendaftarkan endpoint publik goods. /goods dan
// /goods/:slug memakai OptionalAuth() (bukan tanpa middleware sama sekali)
// supaya handler tahu identitas caller BILA kebetulan sedang login — dipakai
// menyembunyikan purchaseUrl/purchaseButtonLabel dari tamu & akun
// Pengunjung/self-registrasi (lihat HandlerImpl.canSeePurchaseLink), tanpa
// mewajibkan login untuk sekadar melihat katalog.
func RegisterPublicRoutes(pub *gin.RouterGroup, h goods_handler.Handler, mw *middlewares.Middleware) {
	pub.GET("/goods", mw.OptionalAuth(), h.PublicList)
	pub.GET("/goods-categories", h.PublicCategories)
	pub.GET("/goods/:slug", mw.OptionalAuth(), h.PublicDetail)
}

// RegisterCMSRoutes mendaftarkan endpoint CMS goods (produk & kategori).
func RegisterCMSRoutes(rg *gin.RouterGroup, h goods_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/goods")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("", mw.RequirePermission(constants.PermGoodsView), h.CMSList)
		g.GET("/:id", mw.RequirePermission(constants.PermGoodsView), h.CMSGet)
		g.POST("", mw.RequirePermission(constants.PermGoodsCreate), h.Create)
		g.PUT("/:id", mw.RequirePermission(constants.PermGoodsUpdate), h.Update)
		g.PATCH("/:id/publish", mw.RequirePermission(constants.PermGoodsPublish), h.Publish)
		g.PATCH("/:id/featured", mw.RequirePermission(constants.PermGoodsUpdate), h.SetFeatured)
		g.DELETE("/:id", mw.RequirePermission(constants.PermGoodsDelete), h.Delete)
	}

	gc := rg.Group("/goods-categories")
	gc.Use(mw.Auth(), mw.RequireVerified())
	{
		gc.GET("", mw.RequirePermission(constants.PermGoodsCategoryView), h.CategoryList)
		gc.POST("", mw.RequirePermission(constants.PermGoodsCategoryCreate), h.CategoryCreate)
		gc.PUT("/:id", mw.RequirePermission(constants.PermGoodsCategoryUpdate), h.CategoryUpdate)
		gc.DELETE("/:id", mw.RequirePermission(constants.PermGoodsCategoryDelete), h.CategoryDelete)
	}
}
