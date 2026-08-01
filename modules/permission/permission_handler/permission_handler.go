// Package permission_handler adalah lapisan presentasi HTTP modul permission.
package permission_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP untuk permission & menu.
type Handler interface {
	ListAll(c *gin.Context)
	Menu(c *gin.Context)
}
