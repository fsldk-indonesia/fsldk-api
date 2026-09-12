package structure

import (
	"fsldk-api/constants"
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
	strGroup.GET("", mw.RequirePermission(constants.PermStructureView), handler.ListCMS)
	strGroup.POST("", mw.RequirePermission(constants.PermStructureCreate), handler.Create)
	strGroup.GET("/:id", mw.RequirePermission(constants.PermStructureView), handler.ShowCMS)
	strGroup.PUT("/:id", mw.RequirePermission(constants.PermStructureUpdate), handler.Update)
	strGroup.DELETE("/:id", mw.RequirePermission(constants.PermStructureDelete), handler.Delete)
	strGroup.POST("/bulk-delete", mw.RequirePermission(constants.PermStructureDelete), handler.BulkDelete)
}
