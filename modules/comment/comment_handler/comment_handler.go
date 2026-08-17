// Package comment_handler adalah lapisan presentasi HTTP modul comment.
package comment_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul comment.
type Handler interface {
	PublicList(c *gin.Context)
	Create(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)
	React(c *gin.Context)
	GifSearch(c *gin.Context)
	GifCategories(c *gin.Context)
	CMSList(c *gin.Context)
	CMSGet(c *gin.Context)
	BulkDelete(c *gin.Context)
}
