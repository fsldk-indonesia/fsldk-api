// Package content_handler adalah lapisan presentasi HTTP modul content.
package content_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul content.
type Handler interface {
	PublicList(c *gin.Context)
	PublicByKey(c *gin.Context)
	PublicProfile(c *gin.Context)
	PublicOrg(c *gin.Context)
	CMSList(c *gin.Context)
	Update(c *gin.Context)
	OrgList(c *gin.Context)
	OrgCreate(c *gin.Context)
	OrgUpdate(c *gin.Context)
	OrgDelete(c *gin.Context)
}
