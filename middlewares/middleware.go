// Package middlewares memuat seluruh middleware HTTP: recovery, CORS,
// autentikasi JWT, gerbang verifikasi email, otorisasi permission, dan rate limit.
package middlewares

import (
	"context"
	"strconv"
	"strings"

	"fsldk-api/base/appctx"
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

// OrgScopeLoader menentukan apakah sebuah organisasi target dapat diakses
// oleh pengguna pemanggil, berdasarkan organisasi asal (cascade) atau
// wildcardTierAccess-nya. Diimplementasikan oleh modul organization.
type OrgScopeLoader interface {
	IsAccessible(ctx context.Context, callerOrganizationID *int64, callerOrganizationTypeCode, wildcardTierAccess string, targetOrganizationID int64) (bool, error)
}

// Middleware menampung dependensi yang dibutuhkan seluruh middleware.
type Middleware struct {
	Token *token.Manager
	Cfg   config.AppConfig
	Perm  PermissionLoader
	Org   OrgScopeLoader
}

// New membuat instance Middleware.
func New(tm *token.Manager, cfg config.AppConfig, perm PermissionLoader, org OrgScopeLoader) *Middleware {
	return &Middleware{Token: tm, Cfg: cfg, Perm: perm, Org: org}
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
		c.Set(constants.CtxOrganizationID, claims.OrganizationID)
		c.Set(constants.CtxOrganizationTypeCode, claims.OrganizationTypeCode)
		c.Set(constants.CtxWildcardTierAccess, claims.WildcardTierAccess)
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

// RequirePermission memastikan role pengguna memiliki salah satu dari kode
// permission yang diberikan (any-match — berguna untuk endpoint yang sama
// dipakai lintas tier dengan permission menu berbeda). Harus dipasang
// setelah Auth().
func (m *Middleware) RequirePermission(codes ...string) gin.HandlerFunc {
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
			for _, code := range codes {
				if p == code {
					c.Set(constants.CtxPermissions, perms)
					c.Next()
					return
				}
			}
		}
		httphelper.Error(c, apperror.Forbidden("Anda tidak memiliki hak akses untuk aksi ini"))
	}
}

// RequireOrganizationScope memastikan organisasi target (dibaca dari path
// atau query param bernama paramName, default ke organisasi asal pengguna
// bila param kosong) berada dalam jangkauan akses pengguna. Harus dipasang
// setelah Auth(). ID organisasi target yang tervalidasi disimpan di
// constants.CtxTargetOrganizationID untuk dipakai handler/service.
func (m *Middleware) RequireOrganizationScope(paramName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		callerOrgID := appctx.OrganizationID(c)
		callerOrgType := appctx.OrganizationTypeCode(c)
		wildcard := appctx.WildcardTierAccess(c)

		raw := c.Param(paramName)
		if raw == "" {
			raw = c.Query(paramName)
		}

		var targetID int64
		if raw == "" {
			if callerOrgID == nil {
				httphelper.Error(c, apperror.Forbidden("Akun Anda tidak terhubung ke organisasi manapun"))
				return
			}
			targetID = *callerOrgID
		} else {
			id, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || id <= 0 {
				httphelper.Error(c, apperror.BadRequest("ID organisasi tidak valid"))
				return
			}
			targetID = id
		}

		ok, err := m.Org.IsAccessible(c.Request.Context(), callerOrgID, callerOrgType, wildcard, targetID)
		if err != nil {
			httphelper.Error(c, apperror.Internal(""))
			return
		}
		if !ok {
			httphelper.Error(c, apperror.Forbidden("Anda tidak memiliki akses ke organisasi ini"))
			return
		}
		c.Set(constants.CtxTargetOrganizationID, targetID)
		c.Next()
	}
}

// LoadPermissions memuat seluruh kode permission milik role pengguna dan
// menyimpannya ke context tanpa pernah menolak request — berbeda dari
// RequirePermission yang menolak bila kode tertentu tidak dimiliki. Dipakai
// pada route "milik-sendiri" yang otorisasinya bercabang antara pemilik
// konten ATAU pemegang permission tertentu (mis. edit/hapus komentar oleh
// moderator — lihat modules/comment).
func (m *Middleware) LoadPermissions() gin.HandlerFunc {
	return func(c *gin.Context) {
		if roleIDVal, ok := c.Get(constants.CtxRoleID); ok {
			if roleID, ok := roleIDVal.(int64); ok {
				if perms, err := m.Perm.RolePermissions(c.Request.Context(), roleID); err == nil {
					c.Set(constants.CtxPermissions, perms)
				}
			}
		}
		c.Next()
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
