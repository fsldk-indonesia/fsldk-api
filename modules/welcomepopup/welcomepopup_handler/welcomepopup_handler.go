// Package welcomepopup_handler adalah lapisan presentasi HTTP modul
// welcomepopup.
package welcomepopup_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul welcomepopup.
type Handler interface {
	Get(c *gin.Context)
	Update(c *gin.Context)
	GetPublic(c *gin.Context)
}
