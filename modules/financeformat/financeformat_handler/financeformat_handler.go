// Package financeformat_handler is the HTTP presentation layer for the financeformat module.
package financeformat_handler

import "github.com/gin-gonic/gin"

// Handler is the HTTP handler contract for the financeformat module.
type Handler interface {
	PublicList(c *gin.Context)
	Download(c *gin.Context)

	CMSList(c *gin.Context)
	CMSGet(c *gin.Context)
	FormatTypes(c *gin.Context)
	Create(c *gin.Context)
	Update(c *gin.Context)
	Publish(c *gin.Context)
	Delete(c *gin.Context)
}
