package subscription

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/subscription/subscription_handler"
	"fsldk-api/modules/subscription/subscription_repository"
	"fsldk-api/modules/subscription/subscription_service"
	"fsldk-api/pkg/mailer"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes registers public and CMS routes for the subscription module.
func RegisterRoutes(api *gin.RouterGroup, pub *gin.RouterGroup, db *gorm.DB, mail mailer.Mailer, frontendURL string, mw *middlewares.Middleware) subscription_service.Service {
	repo := subscription_repository.NewRepository(db)
	svc := subscription_service.NewService(repo, mail, frontendURL)
	handler := subscription_handler.NewHandler(svc)

	// Public endpoints (footer + Hubungi Kami form) with rate limit: 5 requests per 10 minutes.
	pub.POST("/subscribers", middlewares.RateLimit(0.5, 5), handler.Subscribe)
	pub.POST("/subscribers/unsubscribe", middlewares.RateLimit(0.5, 5), handler.Unsubscribe)

	// CMS endpoints
	cms := api.Group("/subscribers")
	cms.Use(mw.Auth(), mw.RequireVerified())
	cms.GET("", mw.RequirePermission(constants.PermSubscriptionView), handler.List)
	cms.GET("/:id", mw.RequirePermission(constants.PermSubscriptionView), handler.Get)
	cms.POST("/bulk", mw.RequirePermission(constants.PermSubscriptionCreate), handler.BulkAdd)
	cms.PUT("/:id", mw.RequirePermission(constants.PermSubscriptionCreate), handler.Update)
	cms.DELETE("/:id", mw.RequirePermission(constants.PermSubscriptionDelete), handler.Delete)
	cms.POST("/bulk-delete", mw.RequirePermission(constants.PermSubscriptionDelete), handler.BulkDelete)

	return svc
}
