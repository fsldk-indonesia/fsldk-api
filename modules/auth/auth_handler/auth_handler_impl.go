package auth_handler

import (
	"fsldk-api/base/apperror"
	"fsldk-api/base/appctx"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/modules/auth/auth_dto"
	"fsldk-api/modules/auth/auth_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct{ svc auth_service.Service }

// NewHandler membuat Handler auth.
func NewHandler(svc auth_service.Service) Handler { return &HandlerImpl{svc: svc} }

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

func (h *HandlerImpl) Register(c *gin.Context) {
	req, ok := bind[auth_dto.RegisterRequest](c)
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

func (h *HandlerImpl) VerifyEmail(c *gin.Context) {
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

func (h *HandlerImpl) ResendVerification(c *gin.Context) {
	if err := h.svc.ResendVerification(c.Request.Context(), appctx.UserID(c)); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Email verifikasi telah dikirim ulang.", nil)
}

func (h *HandlerImpl) Login(c *gin.Context) {
	req, ok := bind[auth_dto.LoginRequest](c)
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

func (h *HandlerImpl) Google(c *gin.Context) {
	req, ok := bind[auth_dto.GoogleRequest](c)
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

func (h *HandlerImpl) Refresh(c *gin.Context) {
	req, ok := bind[auth_dto.RefreshRequest](c)
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

func (h *HandlerImpl) Me(c *gin.Context) {
	res, err := h.svc.Me(c.Request.Context(), appctx.UserID(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", res)
}

// Logout: sesi berbasis JWT stateless; klien cukup menghapus token.
func (h *HandlerImpl) Logout(c *gin.Context) {
	httphelper.Success(c, "Logout berhasil", nil)
}

func (h *HandlerImpl) ChangePassword(c *gin.Context) {
	req, ok := bind[auth_dto.ChangePasswordRequest](c)
	if !ok {
		return
	}
	if err := h.svc.ChangePassword(c.Request.Context(), appctx.UserID(c), req); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Kata sandi berhasil diubah", nil)
}

func (h *HandlerImpl) ForgotPassword(c *gin.Context) {
	req, ok := bind[auth_dto.ForgotPasswordRequest](c)
	if !ok {
		return
	}
	if err := h.svc.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Jika email terdaftar, tautan reset telah dikirim.", nil)
}

func (h *HandlerImpl) ResetPassword(c *gin.Context) {
	req, ok := bind[auth_dto.ResetPasswordRequest](c)
	if !ok {
		return
	}
	if err := h.svc.ResetPassword(c.Request.Context(), req); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Kata sandi berhasil diatur ulang. Silakan login.", nil)
}
