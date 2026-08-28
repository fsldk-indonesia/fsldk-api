// Package catalogbook_handler is the HTTP presentation layer for the catalogbook module.
package catalogbook_handler

import "github.com/gin-gonic/gin"

// Handler is the HTTP handler contract for the catalogbook module.
type Handler interface {
	PublicList(c *gin.Context)
	PublicDetail(c *gin.Context)
	Like(c *gin.Context)
	Categories(c *gin.Context)
	Languages(c *gin.Context)
	AuthorTypes(c *gin.Context)
	AvailabilityTypes(c *gin.Context)

	CMSList(c *gin.Context)
	CMSGet(c *gin.Context)
	Create(c *gin.Context)
	Update(c *gin.Context)
	Publish(c *gin.Context)
	Delete(c *gin.Context)
}
