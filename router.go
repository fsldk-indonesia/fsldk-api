package main

import (
	"net/http"
	"os"

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

	"fsldk-api/modules/dashboard"
	"fsldk-api/modules/dashboard/dashboard_handler"
	"fsldk-api/modules/dashboard/dashboard_repository"
	"fsldk-api/modules/dashboard/dashboard_service"

	"fsldk-api/modules/shortlink"
	"fsldk-api/modules/shortlink/shortlink_handler"
	"fsldk-api/modules/shortlink/shortlink_repository"
	"fsldk-api/modules/shortlink/shortlink_service"

	"fsldk-api/modules/upload"
	"fsldk-api/modules/upload/upload_handler"
	"fsldk-api/modules/upload/upload_service"
	uploadpkg "fsldk-api/pkg/upload"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// setupRouter merangkai seluruh dependensi (dependency injection manual) dan
// mengembalikan gin.Engine yang telah terkonfigurasi. Fungsi ini berperan
// setara dengan hasil generate Google Wire, namun tanpa memerlukan codegen.
func setupRouter(db *gorm.DB, cfg config.AppConfig) *gin.Engine {
	// Infrastruktur & utilitas
	tm := token.NewManager(cfg.JWTSecret, cfg.JWTRefreshSecret, cfg.JWTAccessExpireMinutes, cfg.JWTRefreshExpireMinutes)
	mail := mailer.New(cfg)
	gverify := googleauth.NewVerifier(cfg.GoogleClientID, cfg.GoogleTokenInfoURL)
	uploader := uploadpkg.NewUploader("assets/uploads", cfg.AppURL)

	// Repository (lapisan akses data)
	permRepo := permission_repository.NewRepository(db)
	userRepo := user_repository.NewRepository(db)
	roleRepo := role_repository.NewRepository(db)
	newsRepo := news_repository.NewRepository(db)
	articleRepo := article_repository.NewRepository(db)
	dashRepo := dashboard_repository.NewRepository(db)
	shortlinkRepo := shortlink_repository.NewRepository(db)
	tokenStore := auth_repository.NewTokenStore(db)

	// Service (logika bisnis)
	permSvc := permission_service.NewService(permRepo)
	authSvc := auth_service.NewService(userRepo, roleRepo, permSvc, tm, tokenStore, mail, gverify, cfg)
	userSvc := user_service.NewService(userRepo)
	roleSvc := role_service.NewService(roleRepo)
	newsSvc := news_service.NewService(newsRepo)
	articleSvc := article_service.NewService(articleRepo)
	dashSvc := dashboard_service.NewService(dashRepo)
	shortlinkSvc := shortlink_service.NewService(shortlinkRepo, cfg.FrontendURL)
	uploadSvc := upload_service.NewService(uploader)

	// Handler (presentasi HTTP)
	authH := auth_handler.NewHandler(authSvc)
	permH := permission_handler.NewHandler(permSvc)
	userH := user_handler.NewHandler(userSvc)
	roleH := role_handler.NewHandler(roleSvc)
	newsH := news_handler.NewHandler(newsSvc)
	articleH := article_handler.NewHandler(articleSvc)
	dashH := dashboard_handler.NewHandler(dashSvc)
	shortlinkH := shortlink_handler.NewHandler(shortlinkSvc)
	uploadH := upload_handler.NewHandler(uploadSvc)

	// Middleware bersama (permSvc memenuhi kontrak PermissionLoader)
	mw := middlewares.New(tm, cfg, permSvc)

	// Engine
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	engine.Use(middlewares.Recovery(), middlewares.CORS(cfg))

	// Sajikan berkas gambar yang diunggah lewat modul upload (assets/uploads/*)
	// sebagai berkas statis publik di /uploads/*.
	_ = os.MkdirAll("assets/uploads", 0755)
	engine.Static("/uploads", "./assets/uploads")

	// Endpoint sistem
	engine.GET("/health", func(c *gin.Context) {
		status := "ok"
		if sqlDB, err := db.DB(); err != nil || sqlDB.Ping() != nil {
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

	shortlink.RegisterCMSRoutes(api, shortlinkH, mw)
	shortlink.RegisterResolveRoute(pub, shortlinkH)

	upload.RegisterCMSRoutes(api, uploadH, mw)

	return engine
}
