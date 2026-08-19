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

	Review(c *gin.Context)
	EstablishLevel(c *gin.Context)
	Publish(c *gin.Context)
	Reopen(c *gin.Context)
	Reassess(c *gin.Context)

	ListKaders(c *gin.Context)
	GetKaderCode(c *gin.Context)
	DeactivateKader(c *gin.Context)
	ReassessKader(c *gin.Context)
	ReinstateKader(c *gin.Context)
}
