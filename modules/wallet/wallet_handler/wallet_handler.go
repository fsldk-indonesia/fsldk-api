// Package wallet_handler adalah lapisan presentasi HTTP modul wallet.
package wallet_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul wallet (CMS — tidak ada lagi
// endpoint milik-sendiri, lihat komentar router.go).
type Handler interface {
	CMSBalance(c *gin.Context)
	CMSLedger(c *gin.Context)
}
