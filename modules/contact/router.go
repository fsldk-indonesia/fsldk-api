package contact

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/contact/contact_handler"
	"fsldk-api/modules/contact/contact_repository"
	"fsldk-api/modules/contact/contact_service"
	"fsldk-api/pkg/mailer"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes registers public and CMS routes for the contact module.
func RegisterRoutes(api *gin.RouterGroup, pub *gin.RouterGroup, db *gorm.DB, mail mailer.Mailer, mw *middlewares.Middleware) contact_service.Service {
	repo := contact_repository.NewRepository(db)
	svc := contact_service.NewService(repo, mail)
	handler := contact_handler.NewHandler(svc)

	// Public endpoint with rate limit: 5 requests per 10 minutes (0.5 req/min, burst 5)
	pub.POST("/contact", middlewares.RateLimit(0.5, 5), handler.Send)

	// CMS endpoints
	cms := api.Group("/contact-messages")
	cms.Use(mw.Auth(), mw.RequireVerified())
	cms.GET("", mw.RequirePermission(constants.PermContactView), handler.List)
	cms.GET("/:id", mw.RequirePermission(constants.PermContactView), handler.Show)
	cms.PATCH("/:id/read", mw.RequirePermission(constants.PermContactView), handler.MarkRead)
	cms.POST("/:id/reply", mw.RequirePermission(constants.PermContactView), handler.Reply)
	cms.DELETE("/:id", mw.RequirePermission(constants.PermContactDelete), handler.Delete)

	return svc
}
