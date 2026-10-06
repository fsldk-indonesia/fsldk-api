// Package rapimnas_handler adalah lapisan presentasi HTTP modul rapimnas.
package rapimnas_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul rapimnas.
type Handler interface {
	Get(c *gin.Context)
	Update(c *gin.Context)
	GetPublic(c *gin.Context)
}
