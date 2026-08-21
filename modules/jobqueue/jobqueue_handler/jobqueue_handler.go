// Package jobqueue_handler adalah lapisan presentasi HTTP modul job queue.
package jobqueue_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul job queue.
type Handler interface {
	CMSList(c *gin.Context)
	CMSGet(c *gin.Context)
	CMSStats(c *gin.Context)
	Retry(c *gin.Context)
	Delete(c *gin.Context)
}
