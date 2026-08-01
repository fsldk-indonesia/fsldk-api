package auth

import (
	"fsldk-api/base/apperror"
	"fsldk-api/base/appctx"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"

	"github.com/gin-gonic/gin"
)

// Handler menangani request HTTP untuk modul auth.
type Handler struct{ svc *Service }

// NewHandler membuat Handler auth.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func bind[T any](c *gin.Context) (T, bool) {
	var req T
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return req, false
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return req, false
	}
	return req, true
}

// Register menangani POST /auth/register.
func (h *Handler) Register(c *gin.Context) {
	req, ok := bind[RegisterRequest](c)
	if !ok {
		return
	}
	res, err := h.svc.Register(c.Request.Context(), req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Created(c, res.Message, res)
}

// VerifyEmail menangani GET /auth/email/verify/:token.
func (h *Handler) VerifyEmail(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		httphelper.Error(c, apperror.BadRequest("Token tidak ditemukan"))
		return
	}
	if err := h.svc.VerifyEmail(c.Request.Context(), token); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Email berhasil diverifikasi. Silakan login.", nil)
}

// ResendVerification menangani POST /auth/email/resend.
func (h *Handler) ResendVerification(c *gin.Context) {
	if err := h.svc.ResendVerification(c.Request.Context(), appctx.UserID(c)); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Email verifikasi telah dikirim ulang.", nil)
}

// Login menangani POST /auth/login.
func (h *Handler) Login(c *gin.Context) {
	req, ok := bind[LoginRequest](c)
	if !ok {
		return
	}
	res, err := h.svc.Login(c.Request.Context(), req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Login berhasil", res)
}

// Google menangani POST /auth/google.
func (h *Handler) Google(c *gin.Context) {
	req, ok := bind[GoogleRequest](c)
	if !ok {
		return
	}
	res, err := h.svc.LoginGoogle(c.Request.Context(), req.IDToken, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Login berhasil", res)
}

// Refresh menangani POST /auth/refresh-token.
func (h *Handler) Refresh(c *gin.Context) {
	req, ok := bind[RefreshRequest](c)
	if !ok {
		return
	}
	res, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Token diperbarui", res)
}

// Me menangani GET /auth/me.
func (h *Handler) Me(c *gin.Context) {
	res, err := h.svc.Me(c.Request.Context(), appctx.UserID(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", res)
}

// Logout menangani POST /auth/logout.
// Sesi berbasis JWT bersifat stateless; klien cukup menghapus token.
func (h *Handler) Logout(c *gin.Context) {
	httphelper.Success(c, "Logout berhasil", nil)
}

// ChangePassword menangani POST /auth/change-password.
func (h *Handler) ChangePassword(c *gin.Context) {
	req, ok := bind[ChangePasswordRequest](c)
	if !ok {
		return
	}
	if err := h.svc.ChangePassword(c.Request.Context(), appctx.UserID(c), req); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Kata sandi berhasil diubah", nil)
}

// ForgotPassword menangani POST /auth/forgot-password.
func (h *Handler) ForgotPassword(c *gin.Context) {
	req, ok := bind[ForgotPasswordRequest](c)
	if !ok {
		return
	}
	if err := h.svc.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Jika email terdaftar, tautan reset telah dikirim.", nil)
}

// ResetPassword menangani POST /auth/reset-password.
func (h *Handler) ResetPassword(c *gin.Context) {
	req, ok := bind[ResetPasswordRequest](c)
	if !ok {
		return
	}
	if err := h.svc.ResetPassword(c.Request.Context(), req); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Kata sandi berhasil diatur ulang. Silakan login.", nil)
}
