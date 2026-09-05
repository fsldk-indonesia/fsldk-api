// Package goods_handler adalah lapisan presentasi HTTP modul goods.
package goods_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul goods.
type Handler interface {
	PublicList(c *gin.Context)
	PublicDetail(c *gin.Context)
	PublicCategories(c *gin.Context)

	CMSList(c *gin.Context)
	CMSGet(c *gin.Context)
	Create(c *gin.Context)
	Update(c *gin.Context)
	Publish(c *gin.Context)
	SetFeatured(c *gin.Context)
	Delete(c *gin.Context)

	CategoryList(c *gin.Context)
	CategoryCreate(c *gin.Context)
	CategoryUpdate(c *gin.Context)
	CategoryDelete(c *gin.Context)
}
