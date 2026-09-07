package qrcoderequest_service

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/modules/jobqueue/jobqueue_dto"
	"fsldk-api/modules/jobqueue/jobqueue_model"
	"fsldk-api/modules/qrcode/qrcode_model"
	"fsldk-api/modules/qrcode/qrcoderequest_dto"
	"fsldk-api/modules/qrcode/qrcoderequest_model"
	"fsldk-api/modules/qrcode/qrcoderequest_repository"
	"fsldk-api/modules/setting/setting_model"
	"fsldk-api/pkg/kirimdev"
)

// Warna default selaras tema sistem (lihat qrcode_service).
const (
	defaultForeground = "#16211C"
	defaultBackground = "#FFFFFF"
)

func colorOrDefault(v, def string) string {
	if v = strings.TrimSpace(v); v == "" {
		return def
	}
	return strings.ToUpper(v)
}

func nzs(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// SettingReader is the narrow slice of setting_service.Service this module
// depends on — accept-interfaces idiom, same as shortlinkrequest_service.
type SettingReader interface {
	GetValue(ctx context.Context, group, key string) (string, error)
}

// JobEnqueuer is the narrow slice of jobqueue_service.Service used to send
// notifications. Satisfied automatically by jobqueue_service.Service.
type JobEnqueuer interface {
	Enqueue(ctx context.Context, in jobqueue_dto.EnqueueInput) (int64, error)
}

// WhatsAppMessageResolver is the narrow slice of jobqueue_service.Service used
// to resolve which QR code request an inbound WhatsApp reply refers to —
// reply-threading via tr_whatsapp_message_log.
type WhatsAppMessageResolver interface {
	ResolveCorrelation(ctx context.Context, waMessageID string) (correlationType string, correlationID int64, found bool, err error)
	FindRecentByPhone(ctx context.Context, phone, correlationType string, limit int) ([]int64, error)
}

// sortColumns memetakan field sort yang diizinkan ke kolom database.
var sortColumns = map[string]string{
	"requesterName": "qr.requesterName",
	"status":        "qr.status",
	"createdDate":   "qr.createdDate",
}

var phoneCleanPattern = regexp.MustCompile(`[^0-9]`)

// normalizePhone menormalkan nomor WhatsApp ke format 62xxxxxxxxxx.
func normalizePhone(raw string) string {
	digits := phoneCleanPattern.ReplaceAllString(raw, "")
	if strings.HasPrefix(digits, "0") {
		return "62" + digits[1:]
	}
	return digits
}

// detectIntent membaca maksud balasan WhatsApp (approve/reject) dari tombol
// quick-reply ATAU teks bebas.
func detectIntent(payload kirimdev.InboundWebhookPayload) (action string, ok bool) {
	if payload.Button != nil {
		switch strings.ToLower(strings.TrimSpace(payload.Button.Payload)) {
		case "approve":
			return "approve", true
		case "reject":
			return "reject", true
		}
	}
	if payload.Interactive != nil && payload.Interactive.ButtonReply != nil {
		switch strings.ToLower(strings.TrimSpace(payload.Interactive.ButtonReply.ID)) {
		case "approve":
			return "approve", true
		case "reject":
			return "reject", true
		}
	}
	if payload.Text != nil {
		switch strings.ToLower(strings.TrimSpace(payload.Text.Body)) {
		case "yes", "ya", "approve", "setuju":
			return "approve", true
		case "no", "tidak", "reject", "tolak":
			return "reject", true
		}
	}
	return "", false
}

// ServiceImpl adalah implementasi Service. TIDAK memegang *gorm.DB — transaksi
// lintas tabel (ms_qrcode + ms_qrcode_request) dimiliki repository
// (repo.ApproveTx). Notifikasi lewat jobqueue via JobEnqueuer.
type ServiceImpl struct {
	repo       qrcoderequest_repository.Repository
	jobs       JobEnqueuer
	resolver   WhatsAppMessageResolver
	setting    SettingReader
	apiBaseURL string
}

// NewService membuat Service QR Code request.
func NewService(
	repo qrcoderequest_repository.Repository,
	jobs JobEnqueuer,
	resolver WhatsAppMessageResolver,
	setting SettingReader,
	apiBaseURL string,
) Service {
	return &ServiceImpl{
		repo:       repo,
		jobs:       jobs,
		resolver:   resolver,
		setting:    setting,
		apiBaseURL: strings.TrimRight(apiBaseURL, "/"),
	}
}

func (s *ServiceImpl) imageURL(qrCodeID int64) string {
	if qrCodeID == 0 {
		return ""
	}
	return s.apiBaseURL + "/public/qrcodes/" + strconv.FormatInt(qrCodeID, 10) + "/image"
}

func (s *ServiceImpl) toResponse(m qrcoderequest_model.QRCodeRequest) qrcoderequest_dto.Response {
	note, rejectionReason, reviewedDate := "", "", ""
	if m.Note != nil {
		note = *m.Note
	}
	if m.RejectionReason != nil {
		rejectionReason = *m.RejectionReason
	}
	if m.ReviewedDate != nil {
		reviewedDate = m.ReviewedDate.Format("2006-01-02 15:04:05")
	}
	var qrCodeID int64
	if m.QRCodeID != nil {
		qrCodeID = *m.QRCodeID
	}
	return qrcoderequest_dto.Response{
		QRCodeRequestID:   m.QRCodeRequestID,
		RequesterName:     m.RequesterName,
		RequesterEmail:    m.RequesterEmail,
		RequesterWhatsapp: m.RequesterWhatsapp,
		DestinationURL:    m.DestinationURL,
		ForegroundColor:   m.ForegroundColor,
		BackgroundColor:   m.BackgroundColor,
		CenterIconURL:     nzs(m.CenterIconURL),
		CenterIconKey:     nzs(m.CenterIconKey),
		CaptionText:       nzs(m.CaptionText),
		Note:              note,
		Status:            m.Status,
		QRCodeID:          qrCodeID,
		ImageURL:          s.imageURL(qrCodeID),
		RejectionReason:   rejectionReason,
		ReviewedVia:       m.ReviewedVia,
		ReviewerName:      m.ReviewerName,
		ReviewedDate:      reviewedDate,
		CreatedDate:       m.CreatedDate.Format("2006-01-02 15:04:05"),
	}
}

func (s *ServiceImpl) Submit(ctx context.Context, req qrcoderequest_dto.SubmitRequest) (qrcoderequest_dto.Response, error) {
	req.RequesterWhatsapp = normalizePhone(req.RequesterWhatsapp)
	req.ForegroundColor = colorOrDefault(req.ForegroundColor, defaultForeground)
	req.BackgroundColor = colorOrDefault(req.BackgroundColor, defaultBackground)
	req.CenterIconURL = strings.TrimSpace(req.CenterIconURL)
	req.CenterIconKey = strings.TrimSpace(req.CenterIconKey)
	req.CaptionText = strings.TrimSpace(req.CaptionText)

	newID, err := s.repo.Create(ctx, req)
	if err != nil {
		return qrcoderequest_dto.Response{}, apperror.Internal("Gagal menyimpan permintaan")
	}
	created, err := s.repo.FindByID(ctx, newID)
	if err != nil {
		return qrcoderequest_dto.Response{}, apperror.Internal("")
	}

	// picWhatsapp == "" bukan error — submission tetap sukses meskipun App
	// Settings belum dikonfigurasi.
	picName, picNameErr := s.setting.GetValue(ctx, setting_model.GroupLayanan, setting_model.KeyQRCodePICName)
	picWhatsapp, picWhatsappErr := s.setting.GetValue(ctx, setting_model.GroupLayanan, setting_model.KeyQRCodePICWhatsapp)
	if picNameErr != nil || picWhatsappErr != nil {
		log.Printf("[QRCODE-REQUEST] Submit: gagal membaca setting PIC: %v / %v", picNameErr, picWhatsappErr)
	}
	if picWhatsapp == "" {
		log.Printf("[QRCODE-REQUEST] Submit: PIC WhatsApp belum diisi di App Settings, enqueue WA dilewati")
	} else {
		if _, err := s.jobs.Enqueue(ctx, jobqueue_dto.EnqueueInput{
			Queue: jobqueue_model.QueueWhatsApp, JobType: jobqueue_model.JobTypeWhatsAppTemplate,
			Payload: kirimdev.TemplateMessage{
				ToPhone: picWhatsapp, TemplateName: "qrcode_request_notice",
				Params:         []string{picName, req.RequesterName, req.DestinationURL},
				ButtonPayloads: []string{"approve", "reject"},
			},
			CorrelationType: jobqueue_model.CorrelationTypeQRCodeRequest, CorrelationID: newID,
		}); err != nil {
			log.Printf("[QRCODE-REQUEST] Submit: gagal enqueue WA ke PIC: %v", err)
		}
	}

	return s.toResponse(created), nil
}

func (s *ServiceImpl) PublicPIC(ctx context.Context) (qrcoderequest_dto.PICResponse, error) {
	picName, err := s.setting.GetValue(ctx, setting_model.GroupLayanan, setting_model.KeyQRCodePICName)
	if err != nil {
		return qrcoderequest_dto.PICResponse{}, apperror.Internal("")
	}
	picWhatsapp, err := s.setting.GetValue(ctx, setting_model.GroupLayanan, setting_model.KeyQRCodePICWhatsapp)
	if err != nil {
		return qrcoderequest_dto.PICResponse{}, apperror.Internal("")
	}
	return qrcoderequest_dto.PICResponse{PICName: picName, PICWhatsapp: picWhatsapp}, nil
}

func (s *ServiceImpl) CMSList(ctx context.Context, q dto.ListQuery, status string) ([]qrcoderequest_dto.Response, int, error) {
	rows, total, err := s.repo.List(ctx, qrcoderequest_dto.ListFilter{
		Status:  status,
		Search:  q.Search,
		Limit:   q.Limit,
		Offset:  q.Offset(),
		OrderBy: q.OrderBy(sortColumns, "qr.createdDate DESC"),
	})
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	out := make([]qrcoderequest_dto.Response, 0, len(rows))
	for _, r := range rows {
		out = append(out, s.toResponse(r))
	}
	return out, int(total), nil
}

func (s *ServiceImpl) CMSGet(ctx context.Context, id int64) (qrcoderequest_dto.Response, error) {
	m, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return qrcoderequest_dto.Response{}, apperror.NotFound("Permintaan tidak ditemukan")
	}
	return s.toResponse(m), nil
}

// approveRequest adalah SATU-SATUNYA jalan memproses approve — dipanggil baik
// dari Approve (jalur CMS) maupun HandleWhatsAppReply (jalur WhatsApp).
// Mengembalikan sentinel qrcoderequest_repository.Err* apa adanya (BUKAN
// apperror) supaya kedua pemanggil menerjemahkannya sendiri.
func (s *ServiceImpl) approveRequest(ctx context.Context, id int64, reviewerID *int64, reviewedVia string) (qrcoderequest_dto.Response, error) {
	req, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return qrcoderequest_dto.Response{}, qrcoderequest_repository.ErrNotFound
	}
	if req.Status != qrcoderequest_model.StatusPending {
		return qrcoderequest_dto.Response{}, qrcoderequest_repository.ErrAlreadyProcessed
	}

	// Salin kustomisasi yang dipilih pemohon ke baris ms_qrcode baru; arahnya
	// jadi destinationURL asli (bukan lagi dummy preview).
	newQR := qrcode_model.QRCode{
		DestinationURL:  req.DestinationURL,
		ForegroundColor: colorOrDefault(req.ForegroundColor, defaultForeground),
		BackgroundColor: colorOrDefault(req.BackgroundColor, defaultBackground),
		CenterIconURL:   req.CenterIconURL,
		CenterIconKey:   req.CenterIconKey,
		CaptionText:     req.CaptionText,
	}
	qrCodeID, err := s.repo.ApproveTx(ctx, id, newQR, reviewerID, reviewedVia)
	if err != nil {
		return qrcoderequest_dto.Response{}, err
	}

	// Notifikasi di LUAR transaksi DB, lewat job queue — kegagalan enqueue
	// TIDAK membatalkan approval yang sudah tersimpan.
	s.enqueueApprovedNotifications(ctx, req, qrCodeID)

	updated, ferr := s.repo.FindByID(ctx, id)
	if ferr != nil {
		log.Printf("[QRCODE-REQUEST] approveRequest: request %d disetujui tapi gagal re-fetch untuk response: %v", id, ferr)
		now := time.Now()
		req.Status = qrcoderequest_model.StatusApproved
		req.QRCodeID = &qrCodeID
		req.ReviewedBy = reviewerID
		req.ReviewedVia = reviewedVia
		req.ReviewedDate = &now
		return s.toResponse(req), nil
	}
	return s.toResponse(updated), nil
}

// rejectRequest adalah SATU-SATUNYA jalan memproses reject — simetris dengan
// approveRequest.
func (s *ServiceImpl) rejectRequest(ctx context.Context, id int64, reviewerID *int64, reviewedVia string, reason *string) error {
	req, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return qrcoderequest_repository.ErrNotFound
	}
	if req.Status != qrcoderequest_model.StatusPending {
		return qrcoderequest_repository.ErrAlreadyProcessed
	}
	if err := s.repo.UpdateStatus(ctx, id, qrcoderequest_model.StatusRejected, reviewerID, reviewedVia, reason); err != nil {
		return err
	}
	s.enqueueRejectedNotifications(ctx, req, *reason)
	return nil
}

// Approve adalah pembungkus tipis approveRequest untuk jalur CMS.
func (s *ServiceImpl) Approve(ctx context.Context, id, reviewerID int64) (qrcoderequest_dto.Response, error) {
	rid := reviewerID
	res, err := s.approveRequest(ctx, id, &rid, qrcoderequest_model.ReviewedViaCMS)
	switch {
	case errors.Is(err, qrcoderequest_repository.ErrNotFound):
		return qrcoderequest_dto.Response{}, apperror.NotFound("Permintaan tidak ditemukan")
	case errors.Is(err, qrcoderequest_repository.ErrAlreadyProcessed):
		return qrcoderequest_dto.Response{}, apperror.Conflict("Permintaan sudah diproses sebelumnya")
	case err != nil:
		return qrcoderequest_dto.Response{}, apperror.Internal("Gagal menyetujui permintaan")
	}
	return res, nil
}

// Reject adalah pembungkus tipis rejectRequest untuk jalur CMS.
func (s *ServiceImpl) Reject(ctx context.Context, id, reviewerID int64, reason string) error {
	rid := reviewerID
	err := s.rejectRequest(ctx, id, &rid, qrcoderequest_model.ReviewedViaCMS, &reason)
	switch {
	case errors.Is(err, qrcoderequest_repository.ErrNotFound):
		return apperror.NotFound("Permintaan tidak ditemukan")
	case errors.Is(err, qrcoderequest_repository.ErrAlreadyProcessed):
		return apperror.Conflict("Permintaan sudah diproses sebelumnya")
	case err != nil:
		return apperror.Internal("")
	}
	return nil
}

// resolveTargetRequest menentukan qrCodeRequestID yang dimaksud sebuah balasan
// WhatsApp — reply-threading (context.id) lebih dulu, baru fallback ke
// pencocokan pending-terbaru milik nomor yang sama. TIDAK PERNAH menebak kalau
// ada >1 kandidat ambigu.
func (s *ServiceImpl) resolveTargetRequest(ctx context.Context, payload kirimdev.InboundWebhookPayload) (int64, bool) {
	if payload.Context != nil && payload.Context.ID != "" {
		correlationType, correlationID, found, err := s.resolver.ResolveCorrelation(ctx, payload.Context.ID)
		if err == nil && found && correlationType == jobqueue_model.CorrelationTypeQRCodeRequest {
			if req, ferr := s.repo.FindByID(ctx, correlationID); ferr == nil && req.Status == qrcoderequest_model.StatusPending {
				return correlationID, true
			}
		}
	}

	normalized := normalizePhone(payload.From)
	ids, err := s.resolver.FindRecentByPhone(ctx, normalized, jobqueue_model.CorrelationTypeQRCodeRequest, 20)
	if err != nil || len(ids) == 0 {
		return 0, false
	}
	pending, err := s.repo.FindPendingByIDs(ctx, ids)
	if err != nil || len(pending) != 1 {
		return 0, false // 0 atau >1 kandidat — TIDAK menebak
	}
	return pending[0].QRCodeRequestID, true
}

func (s *ServiceImpl) HandleWhatsAppReply(ctx context.Context, payload kirimdev.InboundWebhookPayload) (WhatsAppReplyOutcome, error) {
	picWhatsapp, _ := s.setting.GetValue(ctx, setting_model.GroupLayanan, setting_model.KeyQRCodePICWhatsapp)
	if picWhatsapp == "" {
		return OutcomeIgnoredNotPIC, nil
	}
	if normalizePhone(payload.From) != normalizePhone(picWhatsapp) {
		return OutcomeIgnoredNotPIC, nil // balasan dari nomor selain PIC diabaikan
	}

	action, ok := detectIntent(payload)
	if !ok {
		return OutcomeIgnoredNoIntent, nil
	}

	requestID, ok := s.resolveTargetRequest(ctx, payload)
	if !ok {
		return OutcomeAmbiguousOrNotFound, nil
	}

	if action == "approve" {
		_, err := s.approveRequest(ctx, requestID, nil, qrcoderequest_model.ReviewedViaWhatsApp)
		switch {
		case errors.Is(err, qrcoderequest_repository.ErrAlreadyProcessed):
			return OutcomeAlreadyProcessed, nil
		case err != nil:
			return OutcomeApproved, err
		}
		return OutcomeApproved, nil
	}

	reason := "Ditolak oleh PIC via WhatsApp (tanpa alasan tertulis)"
	err := s.rejectRequest(ctx, requestID, nil, qrcoderequest_model.ReviewedViaWhatsApp, &reason)
	if errors.Is(err, qrcoderequest_repository.ErrAlreadyProcessed) {
		return OutcomeAlreadyProcessed, nil
	}
	return OutcomeRejected, err
}

func (s *ServiceImpl) enqueueApprovedNotifications(ctx context.Context, req qrcoderequest_model.QRCodeRequest, qrCodeID int64) {
	imageURL := s.imageURL(qrCodeID)
	// Yang dikirim ke pengaju lewat WhatsApp adalah tautan UNDUH gambar QR
	// hasil approve (ukuran cetak 1024px), bukan tautan tujuan.
	downloadURL := imageURL
	if downloadURL != "" {
		downloadURL += "?size=1024"
	}
	if _, err := s.jobs.Enqueue(ctx, jobqueue_dto.EnqueueInput{
		Queue: jobqueue_model.QueueWhatsApp, JobType: jobqueue_model.JobTypeWhatsAppTemplate,
		Payload: kirimdev.TemplateMessage{
			ToPhone: req.RequesterWhatsapp, TemplateName: "qrcode_approved",
			Params: []string{req.RequesterName, downloadURL},
		},
		CorrelationType: jobqueue_model.CorrelationTypeQRCodeRequest, CorrelationID: req.QRCodeRequestID,
	}); err != nil {
		log.Printf("[QRCODE-REQUEST] Approve: gagal enqueue WA ke requester: %v", err)
	}
	if _, err := s.jobs.Enqueue(ctx, jobqueue_dto.EnqueueInput{
		Queue: jobqueue_model.QueueEmail, JobType: jobqueue_model.JobTypeEmailQRCodeApproved,
		Payload:         jobqueue_dto.QRCodeApprovedEmailPayload{ToEmail: req.RequesterEmail, ToName: req.RequesterName, ImageURL: imageURL},
		CorrelationType: jobqueue_model.CorrelationTypeQRCodeRequest, CorrelationID: req.QRCodeRequestID,
	}); err != nil {
		log.Printf("[QRCODE-REQUEST] Approve: gagal enqueue email ke requester: %v", err)
	}
}

func (s *ServiceImpl) enqueueRejectedNotifications(ctx context.Context, req qrcoderequest_model.QRCodeRequest, reason string) {
	if _, err := s.jobs.Enqueue(ctx, jobqueue_dto.EnqueueInput{
		Queue: jobqueue_model.QueueWhatsApp, JobType: jobqueue_model.JobTypeWhatsAppTemplate,
		Payload: kirimdev.TemplateMessage{
			ToPhone: req.RequesterWhatsapp, TemplateName: "qrcode_rejected",
			Params: []string{req.RequesterName, reason},
		},
		CorrelationType: jobqueue_model.CorrelationTypeQRCodeRequest, CorrelationID: req.QRCodeRequestID,
	}); err != nil {
		log.Printf("[QRCODE-REQUEST] Reject: gagal enqueue WA ke requester: %v", err)
	}
	if _, err := s.jobs.Enqueue(ctx, jobqueue_dto.EnqueueInput{
		Queue: jobqueue_model.QueueEmail, JobType: jobqueue_model.JobTypeEmailQRCodeRejected,
		Payload:         jobqueue_dto.QRCodeRejectedEmailPayload{ToEmail: req.RequesterEmail, ToName: req.RequesterName, Reason: reason},
		CorrelationType: jobqueue_model.CorrelationTypeQRCodeRequest, CorrelationID: req.QRCodeRequestID,
	}); err != nil {
		log.Printf("[QRCODE-REQUEST] Reject: gagal enqueue email ke requester: %v", err)
	}
}
