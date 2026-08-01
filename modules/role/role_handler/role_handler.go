// Package role_handler adalah lapisan presentasi HTTP modul role.
package role_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul role.
type Handler interface {
	List(c *gin.Context)
	Get(c *gin.Context)
	Create(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)
	SetPermissions(c *gin.Context)
	Users(c *gin.Context)
}
