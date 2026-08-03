// Package dashboard_handler adalah lapisan presentasi HTTP modul dashboard.
package dashboard_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul dashboard.
type Handler interface {
	Summary(c *gin.Context)
}
