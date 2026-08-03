// Package upload_handler adalah lapisan presentasi HTTP modul upload.
package upload_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul upload.
type Handler interface {
	UploadImage(c *gin.Context)
}
