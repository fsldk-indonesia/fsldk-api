// Package campaign_handler adalah lapisan presentasi HTTP modul campaign.
package campaign_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul campaign (publik, milik-sendiri, CMS).
type Handler interface {
	PublicList(c *gin.Context)
	PublicDetail(c *gin.Context)
	Categories(c *gin.Context)

	MyList(c *gin.Context)
	MyGet(c *gin.Context)
	Create(c *gin.Context)
	Update(c *gin.Context)
	UpdateBeneficiary(c *gin.Context)
	Submit(c *gin.Context)

	CMSList(c *gin.Context)
	CMSGet(c *gin.Context)
	ReviewHistory(c *gin.Context)
	Review(c *gin.Context)
	Publish(c *gin.Context)
	Pause(c *gin.Context)
	Resume(c *gin.Context)
	Archive(c *gin.Context)
}
