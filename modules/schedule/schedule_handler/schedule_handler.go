// Package schedule_handler is the HTTP presentation layer for the schedule module.
package schedule_handler

import "github.com/gin-gonic/gin"

// Handler is the HTTP handler contract for the schedule module.
type Handler interface {
	PublicList(c *gin.Context)

	CMSList(c *gin.Context)
	CMSGet(c *gin.Context)
	Create(c *gin.Context)
	Update(c *gin.Context)
	Publish(c *gin.Context)
	Delete(c *gin.Context)
}
