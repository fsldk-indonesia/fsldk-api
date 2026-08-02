// Package shortlink_handler adalah lapisan presentasi HTTP modul shortlink.
package shortlink_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul shortlink.
type Handler interface {
	List(c *gin.Context)
	Get(c *gin.Context)
	Create(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)
	// Resolve adalah handler publik (tanpa auth) yang mengembalikan URL tujuan
	// dari sebuah kunci shortlink sebagai JSON (bukan redirect langsung) —
	// dipanggil frontend fsldk-web untuk melakukan redirect di sisi browser
	// pada domainnya sendiri (bukan domain backend).
	Resolve(c *gin.Context)
}
