// Package qrcoderequest_handler adalah lapisan presentasi HTTP modul QR Code
// request.
package qrcoderequest_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul QR Code request. Webhook Kirimdev
// TIDAK di sini — satu-satunya route /public/webhooks/kirimdev dimiliki
// shortlinkrequest_handler, yang mem-fan-out balasan ke Service ini.
type Handler interface {
	// Submit adalah handler publik (rate-limited) untuk mengirim permintaan baru.
	Submit(c *gin.Context)
	// PublicPIC adalah handler publik yang mengembalikan info kontak PIC.
	PublicPIC(c *gin.Context)
	CMSList(c *gin.Context)
	CMSGet(c *gin.Context)
	Approve(c *gin.Context)
	Reject(c *gin.Context)
}
