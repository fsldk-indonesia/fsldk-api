// Package dynamicform_handler is the HTTP presentation layer for the dynamicform module.
package dynamicform_handler

import "github.com/gin-gonic/gin"

// Handler is the HTTP handler contract for the dynamicform module.
type Handler interface {
	// public
	PublicGet(c *gin.Context)
	PublicSubmit(c *gin.Context)
	SaveDraft(c *gin.Context)
	StageDraftFile(c *gin.Context)
	RemoveDraftFile(c *gin.Context)

	// CMS — forms
	CMSList(c *gin.Context)
	CMSGet(c *gin.Context)
	Create(c *gin.Context)
	Update(c *gin.Context)
	SetStatus(c *gin.Context)
	Delete(c *gin.Context)
	BulkDelete(c *gin.Context)

	// CMS — builder
	AddField(c *gin.Context)
	UpdateField(c *gin.Context)
	RemoveField(c *gin.Context)
	ReorderFields(c *gin.Context)

	// CMS — rekap / analytics / export
	ListSubmissions(c *gin.Context)
	GetSubmission(c *gin.Context)
	UpdateSubmission(c *gin.Context)
	DeleteSubmission(c *gin.Context)
	ExportCSV(c *gin.Context)
	DeleteResponses(c *gin.Context)
	Analytics(c *gin.Context)

	// CMS — Google Sheets
	GSheetConnect(c *gin.Context)
	GSheetResync(c *gin.Context)
	GSheetDisconnect(c *gin.Context)
}
