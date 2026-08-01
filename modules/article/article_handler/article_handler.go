// Package article_handler adalah lapisan presentasi HTTP modul article.
package article_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul article.
type Handler interface {
	PublicList(c *gin.Context)
	PublicDetail(c *gin.Context)
	Categories(c *gin.Context)
	CMSList(c *gin.Context)
	CMSGet(c *gin.Context)
	Create(c *gin.Context)
	Update(c *gin.Context)
	Publish(c *gin.Context)
	Delete(c *gin.Context)
}
