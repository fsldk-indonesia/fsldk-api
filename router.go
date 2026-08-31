package main

import (
	"net/http"
	"os"
	"time"

	"fsldk-api/base/token"
	"fsldk-api/config"
	"fsldk-api/middlewares"
	"fsldk-api/pkg/goldprice"
	"fsldk-api/pkg/googleauth"
	"fsldk-api/pkg/kirimdev"
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

	"fsldk-api/modules/organization"
	"fsldk-api/modules/organization/organization_handler"
	"fsldk-api/modules/organization/organization_repository"
	"fsldk-api/modules/organization/organization_service"

	"fsldk-api/modules/submission_form"
	"fsldk-api/modules/submission_form/submission_form_handler"
	"fsldk-api/modules/submission_form/submission_form_repository"
	"fsldk-api/modules/submission_form/submission_form_service"

	"fsldk-api/modules/submission"
	"fsldk-api/modules/submission/submission_handler"
	"fsldk-api/modules/submission/submission_repository"
	"fsldk-api/modules/submission/submission_service"

	"fsldk-api/modules/news"
	"fsldk-api/modules/news/news_handler"
	"fsldk-api/modules/news/news_repository"
	"fsldk-api/modules/news/news_service"

	"fsldk-api/modules/article"
	"fsldk-api/modules/article/article_handler"
	"fsldk-api/modules/article/article_repository"
	"fsldk-api/modules/article/article_service"

	"fsldk-api/modules/catalogbook"
	"fsldk-api/modules/catalogbook/catalogbook_handler"
	"fsldk-api/modules/catalogbook/catalogbook_repository"
	"fsldk-api/modules/catalogbook/catalogbook_service"

	"fsldk-api/modules/financeformat"
	"fsldk-api/modules/financeformat/financeformat_handler"
	"fsldk-api/modules/financeformat/financeformat_repository"
	"fsldk-api/modules/financeformat/financeformat_service"

	"fsldk-api/modules/event"
	"fsldk-api/modules/event/event_handler"
	"fsldk-api/modules/event/event_repository"
	"fsldk-api/modules/event/event_service"

	"fsldk-api/modules/comment"
	"fsldk-api/modules/comment/comment_handler"
	"fsldk-api/modules/comment/comment_repository"
	"fsldk-api/modules/comment/comment_service"

	"fsldk-api/modules/dashboard"
	"fsldk-api/modules/dashboard/dashboard_handler"
	"fsldk-api/modules/dashboard/dashboard_repository"
	"fsldk-api/modules/dashboard/dashboard_service"

	"fsldk-api/modules/shortlink"
	"fsldk-api/modules/shortlink/shortlink_handler"
	"fsldk-api/modules/shortlink/shortlink_repository"
	"fsldk-api/modules/shortlink/shortlink_service"
	"fsldk-api/modules/shortlink/shortlinkrequest_handler"
	"fsldk-api/modules/shortlink/shortlinkrequest_repository"
	"fsldk-api/modules/shortlink/shortlinkrequest_service"

	"fsldk-api/modules/setting"
	"fsldk-api/modules/setting/setting_handler"
	"fsldk-api/modules/setting/setting_repository"
	"fsldk-api/modules/setting/setting_service"

	"fsldk-api/modules/jobqueue"
	"fsldk-api/modules/jobqueue/jobqueue_handler"
	"fsldk-api/modules/jobqueue/jobqueue_repository"
	"fsldk-api/modules/jobqueue/jobqueue_service"

	"fsldk-api/modules/upload"
	"fsldk-api/modules/upload/upload_handler"
	"fsldk-api/modules/upload/upload_service"
	uploadpkg "fsldk-api/pkg/upload"

	"fsldk-api/modules/zakat"
	"fsldk-api/modules/zakat/zakat_handler"
	"fsldk-api/modules/zakat/zakat_service"

	"fsldk-api/modules/report"
	"fsldk-api/modules/report/report_handler"
	"fsldk-api/modules/report/report_repository"
	"fsldk-api/modules/report/report_service"
	"fsldk-api/pkg/auditlog"

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
	audit := auditlog.New(db)

	// Repository (lapisan akses data)
	permRepo := permission_repository.NewRepository(db)
	userRepo := user_repository.NewRepository(db)
	roleRepo := role_repository.NewRepository(db)
	orgRepo := organization_repository.NewRepository(db)
	formRepo := submission_form_repository.NewRepository(db)
	subRepo := submission_repository.NewRepository(db)
	newsRepo := news_repository.NewRepository(db)
	articleRepo := article_repository.NewRepository(db)
	catalogbookRepo := catalogbook_repository.NewRepository(db)
	financeformatRepo := financeformat_repository.NewRepository(db)
	eventRepo := event_repository.NewRepository(db)
	dashRepo := dashboard_repository.NewRepository(db)
	shortlinkRepo := shortlink_repository.NewRepository(db)
	reportRepo := report_repository.NewRepository(db)
	shortlinkReqRepo := shortlinkrequest_repository.NewRepository(db)
	settingRepo := setting_repository.NewRepository(db)
	commentRepo := comment_repository.NewRepository(db)
	tokenStore := auth_repository.NewTokenStore(db)

	// Service (logika bisnis)
	permSvc := permission_service.NewService(permRepo)
	orgSvc := organization_service.NewService(orgRepo)
	formSvc := submission_form_service.NewService(formRepo, audit)
	subSvc := submission_service.NewService(subRepo, formRepo, orgRepo, userRepo, roleRepo, orgSvc)
	authSvc := auth_service.NewService(userRepo, roleRepo, permSvc, orgRepo, tm, tokenStore, mail, gverify, cfg)
	userSvc := user_service.NewService(userRepo, orgSvc, audit)
	roleSvc := role_service.NewService(roleRepo)
	dashSvc := dashboard_service.NewService(dashRepo, formRepo, orgSvc)
	shortlinkSvc := shortlink_service.NewService(shortlinkRepo, cfg.FrontendURL)
	settingSvc := setting_service.NewService(settingRepo)
	kirimdevClient := kirimdev.NewClient(cfg.KirimdevAPIKey, cfg.KirimdevPhoneNumberID, cfg.KirimdevBaseURL,
		cfg.KirimdevTemplateLanguage, cfg.KirimdevWebhookSecrets(),
		time.Duration(cfg.KirimdevReplyWindowMinutes)*time.Minute)

	// Job queue (§1b techspec) — dipakai shortlinkrequest_service untuk kirim
	// WhatsApp/email asinkron dengan retry, bukan lagi goroutine langsung.
	jobqueueRepo := jobqueue_repository.NewRepository(db)
	jobqueueSvc := jobqueue_service.NewService(jobqueueRepo, kirimdevClient, mail, cfg)
	jobqueueH := jobqueue_handler.NewHandler(jobqueueSvc)
	workerCount := cfg.JobQueueWorkerCount
	if workerCount <= 0 {
		workerCount = 2
	}
	for i := 0; i < workerCount; i++ {
		go jobqueueSvc.RunWorker(i)
	}
	go jobqueueSvc.RunStuckSweeper()

	// shortlinkReqSvc di-inject jobqueueSvc — satu nilai memenuhi dua interface
	// sempit JobEnqueuer + WhatsAppMessageResolver sekaligus (§6 techspec).
	shortlinkReqSvc := shortlinkrequest_service.NewService(shortlinkReqRepo, shortlinkSvc, jobqueueSvc, jobqueueSvc, settingSvc, cfg.FrontendURL)
	uploadSvc := upload_service.NewService(uploader)
	reportSvc := report_service.NewService(reportRepo, formRepo, orgSvc, audit)
	// Zakat calculator — DB-less; the service wraps the in-memory-cached
	// gold-price client (pkg/goldprice), no repository.
	goldClient := goldprice.NewClient(cfg.ZakatGoldPriceAPIURL, cfg.ZakatGoldPriceFallback, cfg.ZakatGoldPriceCacheMinutes)
	zakatSvc := zakat_service.NewService(goldClient)
	// commentSvc dibuat sebelum newsSvc/articleSvc/catalogbookSvc/eventSvc:
	// keempatnya menerimanya sebagai CommentCleaner untuk membersihkan
	// komentar saat konten induknya dihapus (ms_comment tidak punya FK ke
	// ms_article/ms_news/ms_catalog_book/ms_event, lihat comment techspec §3.1a).
	commentSvc := comment_service.NewService(commentRepo, uploader, cfg.GiphyAPIKey)
	newsSvc := news_service.NewService(newsRepo, commentSvc)
	articleSvc := article_service.NewService(articleRepo, commentSvc)
	catalogbookSvc := catalogbook_service.NewService(catalogbookRepo, uploader, commentSvc)
	eventSvc := event_service.NewService(eventRepo, commentSvc)
	// uploader satisfies FileDeleter; settingSvc satisfies SettingReader for
	// the optional contact-person card on the public page.
	financeformatSvc := financeformat_service.NewService(financeformatRepo, uploader, settingSvc)

	// Handler (presentasi HTTP)
	authH := auth_handler.NewHandler(authSvc)
	permH := permission_handler.NewHandler(permSvc)
	userH := user_handler.NewHandler(userSvc)
	roleH := role_handler.NewHandler(roleSvc)
	orgH := organization_handler.NewHandler(orgSvc)
	formH := submission_form_handler.NewHandler(formSvc)
	subH := submission_handler.NewHandler(subSvc)
	newsH := news_handler.NewHandler(newsSvc)
	articleH := article_handler.NewHandler(articleSvc)
	catalogbookH := catalogbook_handler.NewHandler(catalogbookSvc)
	financeformatH := financeformat_handler.NewHandler(financeformatSvc)
	eventH := event_handler.NewHandler(eventSvc)
	dashH := dashboard_handler.NewHandler(dashSvc)
	shortlinkH := shortlink_handler.NewHandler(shortlinkSvc)
	shortlinkReqH := shortlinkrequest_handler.NewHandler(shortlinkReqSvc, kirimdevClient, jobqueueSvc)
	settingH := setting_handler.NewHandler(settingSvc)
	uploadH := upload_handler.NewHandler(uploadSvc)
	zakatH := zakat_handler.NewHandler(zakatSvc)
	reportH := report_handler.NewHandler(reportSvc)
	commentH := comment_handler.NewHandler(commentSvc)

	// Middleware bersama (permSvc memenuhi kontrak PermissionLoader, orgSvc
	// memenuhi kontrak OrgScopeLoader)
	mw := middlewares.New(tm, cfg, permSvc, orgSvc)

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
	organization.RegisterRoutes(api, orgH, mw)
	submission_form.RegisterRoutes(api, formH, mw)
	submission.RegisterRoutes(api, subH, mw)
	dashboard.RegisterRoutes(api, dashH, mw)
	report.RegisterRoutes(api, reportH, mw)

	news.RegisterPublicRoutes(pub, newsH)
	news.RegisterCMSRoutes(api, newsH, mw)
	article.RegisterPublicRoutes(pub, articleH)
	article.RegisterCMSRoutes(api, articleH, mw)
	catalogbook.RegisterPublicRoutes(pub, catalogbookH)
	catalogbook.RegisterCMSRoutes(api, catalogbookH, mw)
	financeformat.RegisterPublicRoutes(pub, financeformatH)
	financeformat.RegisterCMSRoutes(api, financeformatH, mw)
	event.RegisterPublicRoutes(pub, eventH)
	event.RegisterCMSRoutes(api, eventH, mw)

	shortlink.RegisterCMSRoutes(api, shortlinkH, mw)
	shortlink.RegisterResolveRoute(pub, shortlinkH)
	shortlink.RegisterRequestPublicRoutes(pub, shortlinkReqH)
	shortlink.RegisterRequestCMSRoutes(api, shortlinkReqH, mw)

	setting.RegisterCMSRoutes(api, settingH, mw)
	jobqueue.RegisterCMSRoutes(api, jobqueueH, mw)

	upload.RegisterCMSRoutes(api, uploadH, mw)

	comment.RegisterPublicRoutes(pub, commentH, mw)
	comment.RegisterCMSRoutes(api, commentH, mw)

	zakat.RegisterPublicRoutes(pub, zakatH)

	return engine
}
