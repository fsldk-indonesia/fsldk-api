// Package shortlinkrequest_handler adalah lapisan presentasi HTTP modul
// shortlink request.
package shortlinkrequest_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul shortlink request.
type Handler interface {
	// Submit adalah handler publik (rate-limited) untuk mengirim permintaan baru.
	Submit(c *gin.Context)
	// PublicPIC adalah handler publik yang mengembalikan info kontak PIC
	// (nama + WhatsApp) untuk kartu "Konfirmasi via WhatsApp" di halaman
	// pengajuan — subset read-only App Settings, bukan /settings penuh.
	PublicPIC(c *gin.Context)
	CMSList(c *gin.Context)
	CMSGet(c *gin.Context)
	Approve(c *gin.Context)
	Reject(c *gin.Context)
	// KirimdevWebhook menerima balasan WhatsApp inbound (jalur approval
	// kedua, §1a.5 techspec) — TIDAK LAGI log only, bisa memicu approve/reject
	// lewat Service.HandleWhatsAppReply.
	KirimdevWebhook(c *gin.Context)
}
