// Package organization_handler adalah lapisan presentasi HTTP modul organization.
package organization_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul organization.
type Handler interface {
	Me(c *gin.Context)
	List(c *gin.Context)
	Get(c *gin.Context)
	Children(c *gin.Context)
	Create(c *gin.Context)
	Update(c *gin.Context)
	Deactivate(c *gin.Context)
	Reactivate(c *gin.Context)
}
