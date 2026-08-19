// Package appctx menyediakan pembantu untuk membaca identitas pengguna
// yang disimpan pada gin.Context oleh middleware autentikasi.
package appctx

import (
	"fsldk-api/constants"

	"github.com/gin-gonic/gin"
)

// UserID mengembalikan ID pengguna dari context (0 bila tidak ada).
func UserID(c *gin.Context) int64 {
	if v, ok := c.Get(constants.CtxUserID); ok {
		if id, ok := v.(int64); ok {
			return id
		}
	}
	return 0
}

// RoleID mengembalikan ID role dari context.
func RoleID(c *gin.Context) int64 {
	if v, ok := c.Get(constants.CtxRoleID); ok {
		if id, ok := v.(int64); ok {
			return id
		}
	}
	return 0
}

// RoleName mengembalikan nama role dari context.
func RoleName(c *gin.Context) string {
	if v, ok := c.Get(constants.CtxRoleName); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// Email mengembalikan email pengguna dari context.
func Email(c *gin.Context) string {
	if v, ok := c.Get(constants.CtxUserEmail); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// OrganizationID mengembalikan ID organisasi asal pengguna dari context
// (nil bila akun tidak terikat organisasi spesifik, mis. Super Admin).
func OrganizationID(c *gin.Context) *int64 {
	if v, ok := c.Get(constants.CtxOrganizationID); ok {
		if id, ok := v.(*int64); ok {
			return id
		}
	}
	return nil
}

// OrganizationTypeCode mengembalikan tipe organisasi asal pengguna dari context.
func OrganizationTypeCode(c *gin.Context) string {
	if v, ok := c.Get(constants.CtxOrganizationTypeCode); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// WildcardTierAccess mengembalikan daftar tier organisasi (dipisah koma)
// yang dapat diakses penuh oleh pengguna, mis. "LDK,PUSKOMDA,PUSKOMNAS".
func WildcardTierAccess(c *gin.Context) string {
	if v, ok := c.Get(constants.CtxWildcardTierAccess); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// TargetOrganizationID mengembalikan ID organisasi target yang telah
// divalidasi oleh middleware RequireOrganizationScope.
func TargetOrganizationID(c *gin.Context) int64 {
	if v, ok := c.Get(constants.CtxTargetOrganizationID); ok {
		if id, ok := v.(int64); ok {
			return id
		}
	}
	return 0
}
