// Package auth_handler adalah lapisan presentasi HTTP modul auth.
package auth_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul auth.
type Handler interface {
	Register(c *gin.Context)
	VerifyEmail(c *gin.Context)
	ResendVerification(c *gin.Context)
	Login(c *gin.Context)
	Google(c *gin.Context)
	Refresh(c *gin.Context)
	Me(c *gin.Context)
	Logout(c *gin.Context)
	ChangePassword(c *gin.Context)
	UpdateContact(c *gin.Context)
	ForgotPassword(c *gin.Context)
	ResetPassword(c *gin.Context)
}
