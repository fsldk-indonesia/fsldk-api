// Package contact_handler defines HTTP transport layer for contact inquiries.
package contact_handler

import "github.com/gin-gonic/gin"

// Handler defines HTTP handlers for public and CMS contact endpoints.
type Handler interface {
	Send(c *gin.Context)
	List(c *gin.Context)
	Show(c *gin.Context)
	MarkRead(c *gin.Context)
	Delete(c *gin.Context)
}
