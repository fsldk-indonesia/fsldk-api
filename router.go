package main

import (
	"net/http"

	"fsldk-api/base/token"
	"fsldk-api/config"
	"fsldk-api/middlewares"
	"fsldk-api/pkg/googleauth"
	"fsldk-api/pkg/mailer"

	"fsldk-api/modules/auth"
	"fsldk-api/modules/auth/auth_handler"
	"fsldk-api/modules/auth/auth_repository"
	"fsldk-api/modules/auth/auth_service"

	"fsldk-api/modules/permission"
	"fsldk-api/modules/permission/permission_handler"
	"fsldk-api/modules/permission/permission_repository"
	"fsldk-api/modules/permission/permission_service"

	"fsldk-api/modules/user"
	"fsldk-api/modules/user/user_handler"
	"fsldk-api/modules/user/user_repository"
	"fsldk-api/modules/user/user_service"

	"fsldk-api/modules/role"
	"fsldk-api/modules/role/role_handler"
	"fsldk-api/modules/role/role_repository"
	"fsldk-api/modules/role/role_service"

	"fsldk-api/modules/news"
	"fsldk-api/modules/news/news_handler"
	"fsldk-api/modules/news/news_repository"
	"fsldk-api/modules/news/news_service"

	"fsldk-api/modules/article"
	"fsldk-api/modules/article/article_handler"
	"fsldk-api/modules/article/article_repository"
	"fsldk-api/modules/article/article_service"

	"fsldk-api/modules/content"
	"fsldk-api/modules/content/content_handler"
	"fsldk-api/modules/content/content_repository"
	"fsldk-api/modules/content/content_service"

	"fsldk-api/modules/dashboard"
	"fsldk-api/modules/dashboard/dashboard_handler"
	"fsldk-api/modules/dashboard/dashboard_repository"
	"fsldk-api/modules/dashboard/dashboard_service"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// setupRouter merangkai seluruh dependensi (dependency injection manual) dan
// mengembalikan gin.Engine yang telah terkonfigurasi. Fungsi ini berperan
// setara dengan hasil generate Google Wire, namun tanpa memerlukan codegen.
func setupRouter(db *sqlx.DB, cfg config.AppConfig) *gin.Engine {
	// Infrastruktur & utilitas
	tm := token.NewManager(cfg.JWTSecret, cfg.JWTRefreshSecret, cfg.JWTAccessExpireMinutes, cfg.JWTRefreshExpireMinutes)
	mail := mailer.New(cfg)
	gverify := googleauth.NewVerifier(cfg.GoogleClientID)

	// Repository (lapisan akses data)
	permRepo := permission_repository.NewRepository(db)
	userRepo := user_repository.NewRepository(db)
	roleRepo := role_repository.NewRepository(db)
	newsRepo := news_repository.NewRepository(db)
	articleRepo := article_repository.NewRepository(db)
	contentRepo := content_repository.NewRepository(db)
	dashRepo := dashboard_repository.NewRepository(db)
	tokenStore := auth_repository.NewTokenStore(db)

	// Service (logika bisnis)
	permSvc := permission_service.NewService(permRepo)
	authSvc := auth_service.NewService(userRepo, roleRepo, permSvc, tm, tokenStore, mail, gverify, cfg)
	userSvc := user_service.NewService(userRepo)
	roleSvc := role_service.NewService(roleRepo)
	newsSvc := news_service.NewService(newsRepo)
	articleSvc := article_service.NewService(articleRepo)
	contentSvc := content_service.NewService(contentRepo)
	dashSvc := dashboard_service.NewService(dashRepo)

	// Handler (presentasi HTTP)
	authH := auth_handler.NewHandler(authSvc)
	permH := permission_handler.NewHandler(permSvc)
	userH := user_handler.NewHandler(userSvc)
	roleH := role_handler.NewHandler(roleSvc)
	newsH := news_handler.NewHandler(newsSvc)
	articleH := article_handler.NewHandler(articleSvc)
	contentH := content_handler.NewHandler(contentSvc)
	dashH := dashboard_handler.NewHandler(dashSvc)

	// Middleware bersama (permSvc memenuhi kontrak PermissionLoader)
	mw := middlewares.New(tm, cfg, permSvc)

	// Engine
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	engine.Use(middlewares.Recovery(), middlewares.CORS(cfg))

	// Endpoint sistem
	engine.GET("/health", func(c *gin.Context) {
		status := "ok"
		if err := db.Ping(); err != nil {
			status = "degraded"
		}
		c.JSON(http.StatusOK, gin.H{"status": status, "service": "fsldk-api"})
	})
	engine.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"name": "fsldk-api", "version": "1.0.0", "env": cfg.AppEnv})
	})

	// Grup API
	api := engine.Group("/api/v1")
	pub := api.Group("/public")

	// Registrasi route per modul
	auth.RegisterRoutes(api, authH, mw)
	permission.RegisterRoutes(api, permH, mw)
	user.RegisterRoutes(api, userH, mw)
	role.RegisterRoutes(api, roleH, mw)
	dashboard.RegisterRoutes(api, dashH, mw)

	news.RegisterPublicRoutes(pub, newsH)
	news.RegisterCMSRoutes(api, newsH, mw)
	article.RegisterPublicRoutes(pub, articleH)
	article.RegisterCMSRoutes(api, articleH, mw)
	content.RegisterPublicRoutes(pub, contentH)
	content.RegisterCMSRoutes(api, contentH, mw)

	return engine
}
