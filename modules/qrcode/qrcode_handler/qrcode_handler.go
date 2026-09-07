// Package qrcode_handler adalah lapisan presentasi HTTP modul QR Code.
package qrcode_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul QR Code.
type Handler interface {
	List(c *gin.Context)
	Get(c *gin.Context)
	Create(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)
	// Image adalah handler publik (tanpa auth) yang mengembalikan gambar PNG
	// QR untuk sebuah baris — meng-encode destinationURL langsung.
	Image(c *gin.Context)
}
