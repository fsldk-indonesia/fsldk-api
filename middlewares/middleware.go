// Package middlewares memuat seluruh middleware HTTP: recovery, CORS,
// autentikasi JWT, gerbang verifikasi email, otorisasi permission, dan rate limit.
package middlewares

import (
	"context"
	"strings"

	"fsldk-api/base/apperror"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/token"
	"fsldk-api/config"
	"fsldk-api/constants"

	"github.com/gin-gonic/gin"
)

// PermissionLoader menyediakan daftar kode permission untuk sebuah role.
// Diimplementasikan oleh modul permission agar middleware tidak bergantung
// langsung pada lapisan repository (inversion of dependency).
type PermissionLoader interface {
	RolePermissions(ctx context.Context, roleID int64) ([]string, error)
}

// Middleware menampung dependensi yang dibutuhkan seluruh middleware.
type Middleware struct {
	Token *token.Manager
	Cfg   config.AppConfig
	Perm  PermissionLoader
}

// New membuat instance Middleware.
func New(tm *token.Manager, cfg config.AppConfig, perm PermissionLoader) *Middleware {
	return &Middleware{Token: tm, Cfg: cfg, Perm: perm}
}

// Auth memvalidasi access token dan menyimpan identitas pengguna ke context.
func (m *Middleware) Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := extractBearer(c)
		if raw == "" {
			httphelper.Error(c, apperror.Unauthorized("Token tidak ditemukan"))
			return
		}
		claims, err := m.Token.ParseAccess(raw)
		if err != nil {
			httphelper.Error(c, apperror.Unauthorized("Token tidak valid atau kedaluwarsa"))
			return
		}
		c.Set(constants.CtxUserID, claims.UserID)
		c.Set(constants.CtxUserEmail, claims.Email)
		c.Set(constants.CtxRoleID, claims.RoleID)
		c.Set(constants.CtxRoleName, claims.RoleName)
		c.Set(constants.CtxEmailVerified, claims.EmailVerified)
		c.Next()
	}
}

// OptionalAuth parses the access token the same way Auth() does when one is
// present, but never rejects the request when it's missing or invalid — it
// just proceeds as a guest. Used on public routes that still want to know
// the caller's identity when they happen to be logged in (e.g. the public
// comment thread marking isOwner / the caller's own reactions), without
// requiring login to read them.
func (m *Middleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := extractBearer(c)
		if raw == "" {
			c.Next()
			return
		}
		claims, err := m.Token.ParseAccess(raw)
		if err != nil {
			c.Next()
			return
		}
		c.Set(constants.CtxUserID, claims.UserID)
		c.Set(constants.CtxUserEmail, claims.Email)
		c.Set(constants.CtxRoleID, claims.RoleID)
		c.Set(constants.CtxRoleName, claims.RoleName)
		c.Set(constants.CtxEmailVerified, claims.EmailVerified)
		c.Next()
	}
}

// RequireVerified menolak request bila email pengguna belum terverifikasi.
// Harus dipasang setelah Auth().
func (m *Middleware) RequireVerified() gin.HandlerFunc {
	return func(c *gin.Context) {
		if verified, _ := c.Get(constants.CtxEmailVerified); verified != true {
			httphelper.Error(c, apperror.EmailNotVerified())
			return
		}
		c.Next()
	}
}

// RequirePermission memastikan role pengguna memiliki permission tertentu.
// Harus dipasang setelah Auth().
func (m *Middleware) RequirePermission(code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleIDVal, ok := c.Get(constants.CtxRoleID)
		if !ok {
			httphelper.Error(c, apperror.Unauthorized(""))
			return
		}
		roleID, _ := roleIDVal.(int64)
		perms, err := m.Perm.RolePermissions(c.Request.Context(), roleID)
		if err != nil {
			httphelper.Error(c, apperror.Internal(""))
			return
		}
		for _, p := range perms {
			if p == code {
				c.Set(constants.CtxPermissions, perms)
				c.Next()
				return
			}
		}
		httphelper.Error(c, apperror.Forbidden("Anda tidak memiliki hak akses untuk aksi ini"))
	}
}

func extractBearer(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1])
	}
	return ""
}
