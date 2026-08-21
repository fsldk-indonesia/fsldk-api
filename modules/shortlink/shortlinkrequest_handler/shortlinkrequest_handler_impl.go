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
}

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct {
	svc      shortlinkrequest_service.Service
	kirimdev *kirimdev.Client
	jobs     DeliveryStatusHandler
}

// NewHandler membuat Handler shortlink request.
func NewHandler(svc shortlinkrequest_service.Service, kirimdevClient *kirimdev.Client, jobs DeliveryStatusHandler) Handler {
	return &HandlerImpl{svc: svc, kirimdev: kirimdevClient, jobs: jobs}
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

	// message.status & message.received berbagi endpoint+amplop yang sama,
	// dibedakan lewat header X-Kirim-Event (§12 techspec) — BUKAN dari isi
	// body, karena field JSON-nya sama-sama valid untuk diparsing kosong.
	if c.GetHeader("X-Kirim-Event") == "message.status" {
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
	}

	payload, err := h.kirimdev.ParseInboundWebhook(body)
	if err != nil {
		log.Printf("[KIRIMDEV-WEBHOOK] gagal parse payload: %v — body: %s", err, string(body))
		httphelper.Success(c, "OK", nil) // tetap 200 supaya Kirimdev tidak retry-storm
		return
	}

	outcome, err := h.svc.HandleWhatsAppReply(c.Request.Context(), payload)
	if err != nil {
		log.Printf("[KIRIMDEV-WEBHOOK] gagal proses balasan (outcome=%s): %v", outcome, err)
	} else {
		log.Printf("[KIRIMDEV-WEBHOOK] balasan diproses, outcome=%s", outcome)
	}
	// Selalu 200 KECUALI signature gagal — bukan lagi log only (§1a.5/§6 techspec).
	httphelper.Success(c, "OK", nil)
}
