package structure

import (
	"fsldk-api/middlewares"
	"fsldk-api/modules/structure/structure_handler"
	"fsldk-api/modules/structure/structure_repository"
	"fsldk-api/modules/structure/structure_service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes registers the structure module routes.
func RegisterRoutes(api *gin.RouterGroup, pub *gin.RouterGroup, db *gorm.DB, mw *middlewares.Middleware) {
	repo := structure_repository.NewRepository(db)
	svc := structure_service.NewService(repo)
	handler := structure_handler.NewHandler(svc)

	// Public routes
	pub.GET("/structures", handler.ListPublic)

	// CMS routes
	strGroup := api.Group("/structures")
	strGroup.Use(mw.Auth(), mw.RequireVerified())
	strGroup.GET("", mw.RequirePermission("structure.view"), handler.ListCMS)
	strGroup.POST("", mw.RequirePermission("structure.create"), handler.Create)
	strGroup.GET("/:id", mw.RequirePermission("structure.view"), handler.ShowCMS)
	strGroup.PUT("/:id", mw.RequirePermission("structure.update"), handler.Update)
	strGroup.DELETE("/:id", mw.RequirePermission("structure.delete"), handler.Delete)
}
