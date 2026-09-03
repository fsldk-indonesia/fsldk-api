// Package campaign_handler adalah lapisan presentasi HTTP modul campaign.
package campaign_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul campaign (publik & CMS).
type Handler interface {
	PublicList(c *gin.Context)
	PublicDetail(c *gin.Context)
	Categories(c *gin.Context)

	Create(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)

	CMSList(c *gin.Context)
	CMSListLite(c *gin.Context)
	CMSGet(c *gin.Context)
	Publish(c *gin.Context)
	Pause(c *gin.Context)
	Resume(c *gin.Context)
	Archive(c *gin.Context)
}
