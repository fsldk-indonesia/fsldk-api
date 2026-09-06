package structure_handler

import "github.com/gin-gonic/gin"

// Handler is the HTTP handler contract for structures.
type Handler interface {
	ListPublic(c *gin.Context)
	ListCMS(c *gin.Context)
	ShowCMS(c *gin.Context)
	Create(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)
}
