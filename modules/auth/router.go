package auth

import (
	"fsldk-api/middlewares"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes mendaftarkan seluruh endpoint modul auth.
// Rate limit disesuaikan dengan TechSpec bagian 27.2.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, mw *middlewares.Middleware) {
	g := rg.Group("/auth")
	{
		g.POST("/register", middlewares.RateLimit(0.5, 5), h.Register) // 5 / 10 menit
		g.POST("/login", middlewares.RateLimit(5, 5), h.Login)
		g.POST("/google", middlewares.RateLimit(10, 10), h.Google)
		g.GET("/email/verify/:token", middlewares.RateLimit(6, 6), h.VerifyEmail)
		g.POST("/forgot-password", middlewares.RateLimit(5, 5), h.ForgotPassword)
		g.POST("/reset-password", middlewares.RateLimit(6, 6), h.ResetPassword)

		// Endpoint terautentikasi (tidak mensyaratkan email terverifikasi).
		auth := g.Group("")
		auth.Use(mw.Auth())
		{
			auth.POST("/logout", h.Logout)
			auth.POST("/refresh-token", h.Refresh)
			auth.GET("/me", h.Me)
			auth.POST("/email/resend", middlewares.RateLimit(6, 6), h.ResendVerification)
			auth.POST("/change-password", h.ChangePassword)
		}
	}
}
