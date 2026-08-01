package middlewares

import (
	"log"

	"fsldk-api/base/apperror"
	"fsldk-api/base/httphelper"

	"github.com/gin-gonic/gin"
)

// Recovery menangkap panic agar server tidak crash dan mengembalikan response
// 500 yang rapi (tanpa membocorkan stack trace ke klien).
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[PANIC RECOVERED] %v — %s %s", r, c.Request.Method, c.Request.URL.Path)
				httphelper.Error(c, apperror.Internal(""))
			}
		}()
		c.Next()
	}
}
