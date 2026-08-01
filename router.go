package main

import (
	"net/http"

	"fsldk-api/base/token"
	"fsldk-api/config"
	"fsldk-api/middlewares"
	"fsldk-api/modules/article"
	"fsldk-api/modules/auth"
	"fsldk-api/modules/content"
	"fsldk-api/modules/dashboard"
	"fsldk-api/modules/news"
	"fsldk-api/modules/permission"
	"fsldk-api/modules/role"
	"fsldk-api/modules/user"
	"fsldk-api/pkg/googleauth"
	"fsldk-api/pkg/mailer"

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

	// Repository
	permRepo := permission.NewRepository(db)
	userRepo := user.NewRepository(db)
	roleRepo := role.NewRepository(db)
	newsRepo := news.NewRepository(db)
	articleRepo := article.NewRepository(db)
	contentRepo := content.NewRepository(db)
	tokenStore := auth.NewTokenStore(db)

	// Service
	permSvc := permission.NewService(permRepo)
	authSvc := auth.NewService(userRepo, roleRepo, permSvc, tm, tokenStore, mail, gverify, cfg)
	userSvc := user.NewService(userRepo)
	roleSvc := role.NewService(roleRepo)
	newsSvc := news.NewService(newsRepo)
	articleSvc := article.NewService(articleRepo)
	contentSvc := content.NewService(contentRepo)
	dashSvc := dashboard.NewService(db)

	// Handler
	authH := auth.NewHandler(authSvc)
	permH := permission.NewHandler(permSvc)
	userH := user.NewHandler(userSvc)
	roleH := role.NewHandler(roleSvc)
	newsH := news.NewHandler(newsSvc)
	articleH := article.NewHandler(articleSvc)
	contentH := content.NewHandler(contentSvc)
	dashH := dashboard.NewHandler(dashSvc)

	// Middleware bersama
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
