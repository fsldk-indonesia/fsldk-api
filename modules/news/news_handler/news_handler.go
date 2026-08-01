// Package news_handler adalah lapisan presentasi HTTP modul news.
package news_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul news (publik & CMS).
type Handler interface {
	PublicList(c *gin.Context)
	PublicDetail(c *gin.Context)
	PublicFeatured(c *gin.Context)
	Categories(c *gin.Context)
	CMSList(c *gin.Context)
	CMSGet(c *gin.Context)
	Create(c *gin.Context)
	Update(c *gin.Context)
	Publish(c *gin.Context)
	Featured(c *gin.Context)
	Delete(c *gin.Context)
}
