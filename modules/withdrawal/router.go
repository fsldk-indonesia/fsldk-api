// Package withdrawal merangkai routing modul withdrawal (CMS, callback).
package withdrawal

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/withdrawal/withdrawal_handler"

	"github.com/gin-gonic/gin"
)

// RegisterCMSRoutes mendaftarkan seluruh endpoint withdrawal — TIDAK ada
// lagi endpoint milik-sendiri (revisi 2026-09-01): mengajukan/membatalkan/
// memverifikasi keamanan withdrawal kini murni permission-gated
// (kantong_amal.withdrawal.request), siapapun dengan akses boleh
// menindaklanjuti withdrawal campaign manapun — bukan lagi harus pemilik
// campaign. Tidak ada lagi aksi approve/reject terpisah (maker-checker
// dihapus, revisi 2026-08-30); PermWithdrawalApprove murni menggerbang
// siapa yang bisa melihat/mengelola daftar withdrawal ("Penarikan" di
// sidebar), PermWithdrawalProcess menggerbang aksi pencairan sungguhan.
func RegisterCMSRoutes(rg *gin.RouterGroup, h withdrawal_handler.Handler, mw *middlewares.Middleware) {
	rg.POST("/campaigns/:id/withdrawals", mw.Auth(), mw.RequireVerified(), mw.RequirePermission(constants.PermWithdrawalRequest), h.Request)
	rg.GET("/transfer/banks", mw.Auth(), mw.RequireVerified(), h.ListBanks)
	rg.POST("/transfer/inquiry", mw.Auth(), mw.RequireVerified(), h.Inquiry)

	g := rg.Group("/withdrawals")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("", mw.RequirePermission(constants.PermWithdrawalApprove), h.CMSList)
		g.POST("/:id/cancel", mw.RequirePermission(constants.PermWithdrawalRequest), h.Cancel)
		// Rate limit 5/5menit (§12.9) — reuse pola ldksyahid-app untuk percobaan
		// security verification. Penjamin sesungguhnya tetap attemptCount di
		// tr_otp_challenge (per-user); ini lapisan tambahan per-IP.
		g.POST("/:id/security-verify/otp", middlewares.RateLimit(1, 5), mw.RequirePermission(constants.PermWithdrawalRequest), h.RequestSecurityOtp)
		g.POST("/:id/security-verify", middlewares.RateLimit(1, 5), mw.RequirePermission(constants.PermWithdrawalRequest), h.VerifySecurity)
		g.POST("/:id/process", mw.RequirePermission(constants.PermWithdrawalProcess), h.Process)
	}
}

// RegisterCallbackRoutes mendaftarkan webhook disbursement provider — bukan
// di bawah /public (server-to-server), diamankan URL path secret (bukan
// JWT/signature, lihat withdrawal_service.ProcessCallback).
func RegisterCallbackRoutes(api *gin.RouterGroup, h withdrawal_handler.Handler, mw *middlewares.Middleware) {
	api.POST("/withdrawals/callback/:secret",
		middlewares.IPAllowlist(mw.Cfg.AllowedBisatopupIPsCrowdfunding()),
		middlewares.RateLimit(60, 10),
		h.Callback)
}
