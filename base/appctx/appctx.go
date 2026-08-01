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
