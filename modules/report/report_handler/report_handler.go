// Package report_handler adalah lapisan presentasi HTTP modul report.
package report_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul report.
type Handler interface {
	ExportSubmissions(c *gin.Context)
}
