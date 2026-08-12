// Package comment merangkai routing modul comment (publik & non-publik).
package comment

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/comment/comment_handler"

	"github.com/gin-gonic/gin"
)

// RegisterPublicRoutes mendaftarkan endpoint publik komentar. Bisa diakses
// tanpa login (guest), tapi tetap memakai OptionalAuth supaya isOwner &
// reaksi milik-sendiri terisi benar kalau pemanggilnya kebetulan sedang
// login — tanpa route ini jadi mewajibkan login untuk sekadar membaca.
func RegisterPublicRoutes(pub *gin.RouterGroup, h comment_handler.Handler, mw *middlewares.Middleware) {
	pub.GET("/comments", mw.OptionalAuth(), h.PublicList) // ?contentType=article&contentID=123
}

// RegisterCMSRoutes mendaftarkan endpoint komentar non-publik: aksi milik-sendiri
// (create/update/delete/react, cukup login+verified) dan moderasi admin
// (list/detail/bulk-delete, butuh permission comment.view/comment.delete).
// "CMS" di sini adalah nama baku "grup route non-publik" yang dipakai setiap
// modul di project ini — bukan berarti route-nya khusus staff.
func RegisterCMSRoutes(rg *gin.RouterGroup, h comment_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/comments")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.POST("", h.Create)
		g.PUT("/:id", h.Update)
		g.DELETE("/:id", h.Delete) // owner OR comment.delete — checked in service, see Delete
		g.POST("/:id/react", h.React)
		g.GET("/gif-search", h.GifSearch)
		g.GET("/gif-categories", h.GifCategories)

		g.GET("", mw.RequirePermission(constants.PermCommentView), h.CMSList)
		g.GET("/:id", mw.RequirePermission(constants.PermCommentView), h.CMSGet)
		g.POST("/bulk-delete", mw.RequirePermission(constants.PermCommentDelete), h.BulkDelete)
	}
}
