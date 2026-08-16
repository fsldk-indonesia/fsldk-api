// Package submission_form_handler adalah lapisan presentasi HTTP modul submission_form.
package submission_form_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP form builder.
type Handler interface {
	ListForms(c *gin.Context)
	CreateForm(c *gin.Context)
	GetForm(c *gin.Context)

	CreateVersion(c *gin.Context)
	GetVersion(c *gin.Context)
	PublishVersion(c *gin.Context)

	CreateSection(c *gin.Context)
	UpdateSection(c *gin.Context)
	DeleteSection(c *gin.Context)

	CreateField(c *gin.Context)
	UpdateField(c *gin.Context)
	DeleteField(c *gin.Context)

	CreateOption(c *gin.Context)
	UpdateOption(c *gin.Context)
	DeleteOption(c *gin.Context)
}
