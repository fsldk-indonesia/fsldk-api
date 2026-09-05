// Package subscription_handler defines HTTP transport layer for newsletter subscriptions.
package subscription_handler

import "github.com/gin-gonic/gin"

// Handler defines HTTP handlers for public and CMS subscription endpoints.
type Handler interface {
	Subscribe(c *gin.Context)
	Unsubscribe(c *gin.Context)
	List(c *gin.Context)
	Get(c *gin.Context)
	BulkAdd(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)
	BulkDelete(c *gin.Context)
}
