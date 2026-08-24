// Package withdrawal_handler adalah lapisan presentasi HTTP modul withdrawal.
package withdrawal_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul withdrawal (milik-sendiri, CMS, publik).
type Handler interface {
	Request(c *gin.Context)
	MyList(c *gin.Context)
	Cancel(c *gin.Context)
	RequestSecurityOtp(c *gin.Context)
	VerifySecurity(c *gin.Context)

	CMSList(c *gin.Context)
	Approve(c *gin.Context)
	Reject(c *gin.Context)
	Process(c *gin.Context)

	Callback(c *gin.Context)

	Inquiry(c *gin.Context)
	ListBanks(c *gin.Context)
}
