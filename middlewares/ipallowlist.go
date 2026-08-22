package middlewares

import (
	"fsldk-api/base/apperror"
	"fsldk-api/base/httphelper"

	"github.com/gin-gonic/gin"
)

// IPAllowlist menolak request yang IP-nya tidak ada dalam daftar allowed.
// allowed kosong berarti seluruh IP diizinkan — dipakai endpoint callback
// provider pihak ketiga yang IP allowlist-nya bersifat opsional (lihat
// BISATOPUP_ALLOWED_IPS_CROWDFUNDING).
func IPAllowlist(allowed []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(allowed) == 0 {
			c.Next()
			return
		}
		ip := c.ClientIP()
		for _, a := range allowed {
			if a == ip {
				c.Next()
				return
			}
		}
		httphelper.Error(c, apperror.Forbidden("IP tidak diizinkan mengakses endpoint ini"))
	}
}
