// Package event_handler is the HTTP presentation layer for the event module.
package event_handler

import "github.com/gin-gonic/gin"

// Handler is the HTTP handler contract for events.
type Handler interface {
	ListPublic(c *gin.Context)
	ShowPublic(c *gin.Context)
	ListCMS(c *gin.Context)
	ShowCMS(c *gin.Context)
	Create(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)
}
