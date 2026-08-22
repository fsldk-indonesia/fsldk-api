// Package donation_handler adalah lapisan presentasi HTTP modul donation.
package donation_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul donation (publik, milik-sendiri, CMS).
type Handler interface {
	Create(c *gin.Context)
	Detail(c *gin.Context)
	Status(c *gin.Context)
	Callback(c *gin.Context)

	MyList(c *gin.Context)

	CMSList(c *gin.Context)
}
