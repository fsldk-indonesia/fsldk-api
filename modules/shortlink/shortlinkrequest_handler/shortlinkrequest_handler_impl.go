package shortlinkrequest_handler

import (
	"context"
	"io"
	"log"
	"strconv"

	"fsldk-api/base/appctx"
	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/modules/qrcode/qrcoderequest_service"
	"fsldk-api/modules/shortlink/shortlinkrequest_dto"
	"fsldk-api/modules/shortlink/shortlinkrequest_service"
	"fsldk-api/pkg/kirimdev"

	"github.com/gin-gonic/gin"
)

// DeliveryStatusHandler adalah slice sempit jobqueue_service.Service yang
// dipakai HandlerImpl memproses event webhook "message.status" (§12
// techspec) — accept-interface idiom yang sama seperti SettingReader/
// JobEnqueuer di shortlinkrequest_service, dipenuhi otomatis oleh
// jobqueue_service.Service.
type DeliveryStatusHandler interface {
	HandleDeliveryStatus(ctx context.Context, waMessageID, status, errorDetail string) error
	// HandleMessageSent memproses event "message.sent" — backfill wamid asli
	// begitu Meta menetapkannya (§1a.5 techspec, lihat kirimdev.ParseMessageSentWebhook).
	HandleMessageSent(ctx context.Context, kirimdevMessageID, wamid string) error
}

// QRCodeReplyHandler adalah slice sempit qrcoderequest_service.Service yang
// dipakai HandlerImpl mem-fan-out balasan WhatsApp inbound ke modul QR Code
// request — Kirimdev hanya mem-POST ke satu URL webhook, jadi handler ini
// mencoba modul shortlink dulu lalu jatuh ke modul QR Code bila balasannya
// bukan milik shortlink. Setiap Service memfilter berdasar CorrelationType-nya
// sendiri sehingga urutan pemanggilan aman.
type QRCodeReplyHandler interface {
	HandleWhatsAppReply(ctx context.Context, payload kirimdev.InboundWebhookPayload) (qrcoderequest_service.WhatsAppReplyOutcome, error)
}

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct {
	svc      shortlinkrequest_service.Service
	kirimdev *kirimdev.Client
	jobs     DeliveryStatusHandler
	qrReply  QRCodeReplyHandler
}

// NewHandler membuat Handler shortlink request.
func NewHandler(svc shortlinkrequest_service.Service, kirimdevClient *kirimdev.Client, jobs DeliveryStatusHandler, qrReply QRCodeReplyHandler) Handler {
	return &HandlerImpl{svc: svc, kirimdev: kirimdevClient, jobs: jobs, qrReply: qrReply}
}

func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return id, true
}

func (h *HandlerImpl) Submit(c *gin.Context) {
	var req shortlinkrequest_dto.SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	res, err := h.svc.Submit(c.Request.Context(), req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Created(c, "Permintaan terkirim, kami akan mengabari lewat WhatsApp & email begitu diproses", res)
}

func (h *HandlerImpl) PublicPIC(c *gin.Context) {
	res, err := h.svc.PublicPIC(c.Request.Context())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", res)
}

func (h *HandlerImpl) CMSList(c *gin.Context) {
	q := dto.ParseListQuery(c)
	status := c.Query("status")
	data, total, err := h.svc.CMSList(c.Request.Context(), q, status)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", httphelper.BuildPagination(c, data, total, q.Page, q.Limit))
}

func (h *HandlerImpl) CMSGet(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	res, err := h.svc.CMSGet(c.Request.Context(), id)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", res)
}

func (h *HandlerImpl) Approve(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	res, err := h.svc.Approve(c.Request.Context(), id, appctx.UserID(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Permintaan disetujui, shortlink berhasil dibuat", res)
}

func (h *HandlerImpl) Reject(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req shortlinkrequest_dto.RejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	if err := h.svc.Reject(c.Request.Context(), id, appctx.UserID(c), req.RejectionReason); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Permintaan ditolak", nil)
}

func (h *HandlerImpl) KirimdevWebhook(c *gin.Context) {
	body, _ := io.ReadAll(c.Request.Body)
	signature := c.GetHeader("X-Kirim-Signature")
	if !h.kirimdev.VerifyWebhookSignature(body, signature) {
		httphelper.Error(c, apperror.Unauthorized("Signature tidak valid"))
		return
	}

	// message.status, message.sent & message.received berbagi endpoint yang
	// sama, dibedakan lewat header X-Kirim-Event (§12 techspec) — BUKAN dari
	// isi body, karena amplopnya bisa sama-sama valid diparsing kosong.
	switch c.GetHeader("X-Kirim-Event") {
	case "message.status":
		statuses, err := h.kirimdev.ParseDeliveryStatusWebhook(body)
		if err != nil {
			log.Printf("[KIRIMDEV-WEBHOOK] gagal parse status pengiriman: %v — body: %s", err, string(body))
			httphelper.Success(c, "OK", nil) // tetap 200 supaya Kirimdev tidak retry-storm
			return
		}
		for _, st := range statuses {
			if st.Status != "failed" {
				continue // sent/delivered/read cuma informasional, job sudah 'completed' duluan
			}
			if err := h.jobs.HandleDeliveryStatus(c.Request.Context(), st.WAMessageID, st.Status, st.ErrorDetail); err != nil {
				log.Printf("[KIRIMDEV-WEBHOOK] gagal catat kegagalan pengiriman waMessageID=%s: %v", st.WAMessageID, err)
			}
		}
		httphelper.Success(c, "OK", nil)
		return
	case "message.sent":
		kirimdevMessageID, wamid, err := h.kirimdev.ParseMessageSentWebhook(body)
		if err != nil {
			log.Printf("[KIRIMDEV-WEBHOOK] gagal parse message.sent: %v — body: %s", err, string(body))
			httphelper.Success(c, "OK", nil)
			return
		}
		if err := h.jobs.HandleMessageSent(c.Request.Context(), kirimdevMessageID, wamid); err != nil {
			log.Printf("[KIRIMDEV-WEBHOOK] gagal backfill wamid (kirimdevMessageID=%s, wamid=%s): %v", kirimdevMessageID, wamid, err)
		}
		httphelper.Success(c, "OK", nil)
		return
	}

	payload, err := h.kirimdev.ParseInboundWebhook(body)
	if err != nil {
		log.Printf("[KIRIMDEV-WEBHOOK] gagal parse payload: %v — body: %s", err, string(body))
		httphelper.Success(c, "OK", nil) // tetap 200 supaya Kirimdev tidak retry-storm
		return
	}

	outcome, err := h.svc.HandleWhatsAppReply(c.Request.Context(), payload)
	if err != nil {
		log.Printf("[KIRIMDEV-WEBHOOK] gagal proses balasan shortlink (outcome=%s): %v", outcome, err)
	} else {
		log.Printf("[KIRIMDEV-WEBHOOK] balasan shortlink diproses, outcome=%s", outcome)
	}

	// Fan-out ke modul QR Code request bila balasan ini bukan milik shortlink
	// (nomor PIC beda / tidak ada request pending yang cocok). qrcoderequest
	// memfilter sendiri berdasar CorrelationTypeQRCodeRequest, jadi memanggilnya
	// untuk balasan yang jelas milik shortlink pun tidak berefek.
	if h.qrReply != nil &&
		(outcome == shortlinkrequest_service.OutcomeIgnoredNotPIC || outcome == shortlinkrequest_service.OutcomeAmbiguousOrNotFound) {
		qrOutcome, qrErr := h.qrReply.HandleWhatsAppReply(c.Request.Context(), payload)
		if qrErr != nil {
			log.Printf("[KIRIMDEV-WEBHOOK] gagal proses balasan qrcode (outcome=%s): %v", qrOutcome, qrErr)
		} else {
			log.Printf("[KIRIMDEV-WEBHOOK] balasan qrcode diproses, outcome=%s", qrOutcome)
		}
	}

	// Selalu 200 KECUALI signature gagal — bukan lagi log only (§1a.5/§6 techspec).
	httphelper.Success(c, "OK", nil)
}
