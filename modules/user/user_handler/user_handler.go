// Package user_handler adalah lapisan presentasi HTTP modul user.
package user_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul user.
type Handler interface {
	List(c *gin.Context)
	Get(c *gin.Context)
	Create(c *gin.Context)
	Update(c *gin.Context)
	SetStatus(c *gin.Context)
	Delete(c *gin.Context)
}
