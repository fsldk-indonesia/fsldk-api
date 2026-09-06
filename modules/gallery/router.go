package gallery

import (
	"fsldk-api/middlewares"
	"fsldk-api/modules/gallery/gallery_handler"
	"fsldk-api/modules/gallery/gallery_repository"
	"fsldk-api/modules/gallery/gallery_service"
	uploadpkg "fsldk-api/pkg/upload"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes registers the gallery module public and CMS routes.
func RegisterRoutes(api *gin.RouterGroup, pub *gin.RouterGroup, db *gorm.DB, uploader *uploadpkg.Uploader, mw *middlewares.Middleware) {
	repo := gallery_repository.NewRepository(db)
	svc := gallery_service.NewService(repo, uploader)
	handler := gallery_handler.NewHandler(svc)

	// Public routes
	pub.GET("/galleries", handler.ListPublic)
	pub.GET("/galleries/:id", handler.ShowPublic)
	pub.GET("/galleries/:id/photos", handler.ListPhotosPublic)

	// CMS routes
	galGroup := api.Group("/galleries")
	galGroup.Use(mw.Auth(), mw.RequireVerified())
	galGroup.GET("", mw.RequirePermission("gallery.view"), handler.ListCMS)
	galGroup.POST("", mw.RequirePermission("gallery.create"), handler.Create)
	galGroup.GET("/:id", mw.RequirePermission("gallery.view"), handler.ShowCMS)
	galGroup.PUT("/:id", mw.RequirePermission("gallery.update"), handler.Update)
	galGroup.DELETE("/:id", mw.RequirePermission("gallery.delete"), handler.Delete)

	// Photo sub-endpoints (CMS)
	galGroup.GET("/:id/photos", mw.RequirePermission("gallery.view"), handler.ListPhotosCMS)
	galGroup.POST("/:id/photos", mw.RequirePermission("gallery.update"), handler.AddPhoto)
	galGroup.PUT("/:id/photos/:photoID", mw.RequirePermission("gallery.update"), handler.UpdatePhoto)
	galGroup.DELETE("/:id/photos/:photoID", mw.RequirePermission("gallery.update"), handler.DeletePhoto)
	galGroup.POST("/:id/photos/reorder", mw.RequirePermission("gallery.update"), handler.ReorderPhotos)
}
