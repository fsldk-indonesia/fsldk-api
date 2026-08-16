// Package submission_handler adalah lapisan presentasi HTTP modul submission.
package submission_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul submission.
type Handler interface {
	Create(c *gin.Context)
	SaveAnswers(c *gin.Context)
	Submit(c *gin.Context)
	Cancel(c *gin.Context)
	List(c *gin.Context)
	Get(c *gin.Context)
}
