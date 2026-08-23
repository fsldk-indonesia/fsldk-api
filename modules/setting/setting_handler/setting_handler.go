// Package setting_handler adalah lapisan presentasi HTTP modul setting.
package setting_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul setting.
type Handler interface {
	List(c *gin.Context)
	Update(c *gin.Context)
}
