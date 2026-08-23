// Package wallet_handler adalah lapisan presentasi HTTP modul wallet.
package wallet_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul wallet (milik-sendiri, CMS).
type Handler interface {
	MyBalance(c *gin.Context)
	MyLedger(c *gin.Context)

	CMSBalance(c *gin.Context)
}
