package shortlinkrequest_service

import (
	"context"
	"errors"
	"log"
	"regexp"
	"strings"
	"time"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/modules/jobqueue/jobqueue_dto"
	"fsldk-api/modules/jobqueue/jobqueue_model"
	"fsldk-api/modules/setting/setting_model"
	"fsldk-api/modules/shortlink/shortlink_service"
	"fsldk-api/modules/shortlink/shortlinkrequest_dto"
	"fsldk-api/modules/shortlink/shortlinkrequest_model"
	"fsldk-api/modules/shortlink/shortlinkrequest_repository"
	"fsldk-api/pkg/kirimdev"
)

// SettingReader is the narrow slice of setting_service.Service this module
// depends on — accepting an interface (not importing setting_service by
// concrete type) avoids a hard cross-module dependency, same idiom as
// CommentCleaner in article_service/news_service/event_service.
type SettingReader interface {
	GetValue(ctx context.Context, group, key string) (string, error)
}

// JobEnqueuer is the narrow slice of jobqueue_service.Service this module
// depends on for sending notifications — same accept-interfaces idiom as
// SettingReader. Satisfied automatically by jobqueue_service.Service.
type JobEnqueuer interface {
	Enqueue(ctx context.Context, in jobqueue_dto.EnqueueInput) (int64, error)
}

// WhatsAppMessageResolver is the narrow slice of jobqueue_service.Service
// used to resolve which shortlink request an inbound WhatsApp reply refers
// to (§1a.5 techspec) — reply-threading via tr_whatsapp_message_log, not a
// fragile per-phone cache pointer like the reference implementation.
type WhatsAppMessageResolver interface {
	ResolveCorrelation(ctx context.Context, waMessageID string) (correlationType string, correlationID int64, found bool, err error)
	FindRecentByPhone(ctx context.Context, phone, correlationType string, limit int) ([]int64, error)
}

// sortColumns memetakan field sort yang diizinkan ke kolom database.
var sortColumns = map[string]string{
	"requesterName": "sr.requesterName",
	"status":        "sr.status",
	"createdDate":   "sr.createdDate",
}

// reservedKeys mencegah requestedKey publik collide dengan path bernama yang
// sudah dipakai frontend Angular — rute redirect shortlink (`:key`) adalah
// catch-all satu-segmen di root, jadi kunci yang sama dengan nama rute lain
// berpotensi ambigu.
var reservedKeys = map[string]bool{
	"login": true, "daftar": true, "cms": true, "berita": true, "artikel": true,
	"event": true, "shortlink": true, "verifikasi-email": true, "lupa-password": true,
	"reset-password": true,
}

var phoneCleanPattern = regexp.MustCompile(`[^0-9]`)

// normalizePhone menormalkan nomor WhatsApp ke format 62xxxxxxxxxx (tanpa
// "+", spasi, atau tanda hubung) — mis. "08xx..." atau "+62xx..." -> "62xx...".
func normalizePhone(raw string) string {
	digits := phoneCleanPattern.ReplaceAllString(raw, "")
	if strings.HasPrefix(digits, "0") {
		return "62" + digits[1:]
	}
	return digits
}

// detectIntent membaca maksud balasan WhatsApp (approve/reject) dari tombol
// quick-reply ATAU teks bebas (§1a.5 techspec).
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

// ServiceImpl adalah implementasi Service. TIDAK memegang *gorm.DB sama
// sekali — transaksi lintas tabel (ms_shortlink + ms_shortlink_request)
// dimiliki repository (repo.ApproveTx), bukan dibuka di service. TIDAK LAGI
// memegang kirimdev/mailer langsung (§6/§8.3 techspec) — notifikasi lewat
// jobqueue_service via JobEnqueuer.
type ServiceImpl struct {
	repo         shortlinkrequest_repository.Repository
	shortlinkSvc shortlink_service.Service
	jobs         JobEnqueuer
	resolver     WhatsAppMessageResolver
	setting      SettingReader
	frontendURL  string
}

// NewService membuat Service shortlink request.
func NewService(
	repo shortlinkrequest_repository.Repository,
	shortlinkSvc shortlink_service.Service,
	jobs JobEnqueuer,
	resolver WhatsAppMessageResolver,
	setting SettingReader,
	frontendURL string,
) Service {
	return &ServiceImpl{
		repo:         repo,
		shortlinkSvc: shortlinkSvc,
		jobs:         jobs,
		resolver:     resolver,
		setting:      setting,
		frontendURL:  strings.TrimRight(frontendURL, "/"),
	}
}

func (s *ServiceImpl) toResponse(m shortlinkrequest_model.ShortLinkRequest) shortlinkrequest_dto.Response {
	requestedKey, note, rejectionReason, reviewedDate := "", "", "", ""
	if m.RequestedKey != nil {
		requestedKey = *m.RequestedKey
	}
	if m.Note != nil {
		note = *m.Note
	}
	if m.RejectionReason != nil {
		rejectionReason = *m.RejectionReason
	}
	if m.ReviewedDate != nil {
		reviewedDate = m.ReviewedDate.Format("2006-01-02 15:04:05")
	}
	var shortLinkID int64
	if m.ShortLinkID != nil {
		shortLinkID = *m.ShortLinkID
	}
	shortURL := ""
	if m.ShortKey != "" {
		shortURL = s.frontendURL + "/" + m.ShortKey
	}
	return shortlinkrequest_dto.Response{
		ShortLinkRequestID: m.ShortLinkRequestID,
		RequesterName:      m.RequesterName,
		RequesterEmail:     m.RequesterEmail,
		RequesterWhatsapp:  m.RequesterWhatsapp,
		DestinationURL:     m.DestinationURL,
		RequestedKey:       requestedKey,
		Note:               note,
		Status:             m.Status,
		ShortLinkID:        shortLinkID,
		ShortKey:           m.ShortKey,
		ShortURL:           shortURL,
		RejectionReason:    rejectionReason,
		ReviewedVia:        m.ReviewedVia,
		ReviewerName:       m.ReviewerName,
		ReviewedDate:       reviewedDate,
		CreatedDate:        m.CreatedDate.Format("2006-01-02 15:04:05"),
	}
}

func (s *ServiceImpl) Submit(ctx context.Context, req shortlinkrequest_dto.SubmitRequest) (shortlinkrequest_dto.Response, error) {
	req.RequesterWhatsapp = normalizePhone(req.RequesterWhatsapp)

	key := strings.TrimSpace(req.RequestedKey)
	if key != "" {
		if reservedKeys[strings.ToLower(key)] {
			return shortlinkrequest_dto.Response{}, apperror.Conflict("Kunci tersebut tidak dapat digunakan")
		}
		exists, err := s.shortlinkSvc.KeyExists(ctx, key)
		if err != nil {
			return shortlinkrequest_dto.Response{}, apperror.Internal("")
		}
		if exists {
			return shortlinkrequest_dto.Response{}, apperror.Conflict("Kunci shortlink sudah dipakai")
		}
	}
	req.RequestedKey = key

	newID, err := s.repo.Create(ctx, req)
	if err != nil {
		return shortlinkrequest_dto.Response{}, apperror.Internal("Gagal menyimpan permintaan")
	}
	created, err := s.repo.FindByID(ctx, newID)
	if err != nil {
		return shortlinkrequest_dto.Response{}, apperror.Internal("")
	}

	// picWhatsapp == "" bukan error — submission tetap sukses meskipun App
	// Settings belum dikonfigurasi (§1a.3/§6 techspec). Error dari GetValue
	// (mis. DB hiccup) tetap di-log terpisah supaya tidak disalahartikan
	// sebagai "belum dikonfigurasi" saat mendiagnosis notifikasi yang hilang.
	picName, picNameErr := s.setting.GetValue(ctx, setting_model.GroupLayanan, setting_model.KeyShortlinkPICName)
	picWhatsapp, picWhatsappErr := s.setting.GetValue(ctx, setting_model.GroupLayanan, setting_model.KeyShortlinkPICWhatsapp)
	if picNameErr != nil || picWhatsappErr != nil {
		log.Printf("[SHORTLINK-REQUEST] Submit: gagal membaca setting PIC: %v / %v", picNameErr, picWhatsappErr)
	}
	if picWhatsapp == "" {
		log.Printf("[SHORTLINK-REQUEST] Submit: PIC WhatsApp belum diisi di App Settings, enqueue WA dilewati")
	} else {
		if _, err := s.jobs.Enqueue(ctx, jobqueue_dto.EnqueueInput{
			Queue: jobqueue_model.QueueWhatsApp, JobType: jobqueue_model.JobTypeWhatsAppTemplate,
			Payload: kirimdev.TemplateMessage{
				ToPhone: picWhatsapp, TemplateName: "shortlink_request_notice",
				Params: []string{picName, req.RequesterName, req.DestinationURL, req.RequestedKey},
				// Urut sesuai tombol QUICK_REPLY template ini: index 0 =
				// "Setujui", index 1 = "Tolak" — payload custom inilah yang
				// dicek detectIntent() saat PIC menekan tombolnya (§1a.5).
				ButtonPayloads: []string{"approve", "reject"},
			},
			CorrelationType: jobqueue_model.CorrelationTypeShortlinkRequest, CorrelationID: newID,
		}); err != nil {
			log.Printf("[SHORTLINK-REQUEST] Submit: gagal enqueue WA ke PIC: %v", err)
		}
	}

	return s.toResponse(created), nil
}

func (s *ServiceImpl) PublicPIC(ctx context.Context) (shortlinkrequest_dto.PICResponse, error) {
	picName, err := s.setting.GetValue(ctx, setting_model.GroupLayanan, setting_model.KeyShortlinkPICName)
	if err != nil {
		return shortlinkrequest_dto.PICResponse{}, apperror.Internal("")
	}
	picWhatsapp, err := s.setting.GetValue(ctx, setting_model.GroupLayanan, setting_model.KeyShortlinkPICWhatsapp)
	if err != nil {
		return shortlinkrequest_dto.PICResponse{}, apperror.Internal("")
	}
	return shortlinkrequest_dto.PICResponse{PICName: picName, PICWhatsapp: picWhatsapp}, nil
}

func (s *ServiceImpl) CMSList(ctx context.Context, q dto.ListQuery, status string) ([]shortlinkrequest_dto.Response, int, error) {
	rows, total, err := s.repo.List(ctx, shortlinkrequest_dto.ListFilter{
		Status:  status,
		Search:  q.Search,
		Limit:   q.Limit,
		Offset:  q.Offset(),
		OrderBy: q.OrderBy(sortColumns, "sr.createdDate DESC"),
	})
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	out := make([]shortlinkrequest_dto.Response, 0, len(rows))
	for _, r := range rows {
		out = append(out, s.toResponse(r))
	}
	return out, int(total), nil
}

func (s *ServiceImpl) CMSGet(ctx context.Context, id int64) (shortlinkrequest_dto.Response, error) {
	m, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return shortlinkrequest_dto.Response{}, apperror.NotFound("Permintaan tidak ditemukan")
	}
	return s.toResponse(m), nil
}

// approveRequest adalah SATU-SATUNYA jalan memproses approve — dipanggil
// baik dari Approve (jalur CMS) maupun HandleWhatsAppReply (jalur WhatsApp,
// §1a.5). Mengembalikan sentinel shortlinkrequest_repository.Err* apa adanya
// (BUKAN apperror) supaya kedua pemanggil bisa menerjemahkannya sendiri
// sesuai konteks masing-masing (HTTP response vs webhook outcome, §6/§7).
func (s *ServiceImpl) approveRequest(ctx context.Context, id int64, reviewerID *int64, reviewedVia string) (shortlinkrequest_dto.Response, error) {
	req, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return shortlinkrequest_dto.Response{}, shortlinkrequest_repository.ErrNotFound
	}
	if req.Status != shortlinkrequest_model.StatusPending {
		return shortlinkrequest_dto.Response{}, shortlinkrequest_repository.ErrAlreadyProcessed
	}

	key := ""
	if req.RequestedKey != nil && *req.RequestedKey != "" {
		exists, kerr := s.shortlinkSvc.KeyExists(ctx, *req.RequestedKey)
		if kerr != nil {
			return shortlinkrequest_dto.Response{}, kerr
		}
		if exists {
			return shortlinkrequest_dto.Response{}, shortlinkrequest_repository.ErrKeyCollision
		}
		key = *req.RequestedKey
	} else {
		// Cabang ini sekarang cuma tercapai untuk baris ms_shortlink_request
		// LAMA (requestedKey NULL, tersimpan sebelum field ini jadi wajib) —
		// submission baru selalu punya RequestedKey non-nil (§5 techspec).
		generated, gerr := s.shortlinkSvc.GenerateUniqueKey(ctx)
		if gerr != nil {
			return shortlinkrequest_dto.Response{}, gerr
		}
		key = generated
	}

	shortLinkID, err := s.repo.ApproveTx(ctx, id, req.DestinationURL, key, reviewerID, reviewedVia)
	if err != nil {
		return shortlinkrequest_dto.Response{}, err
	}

	// Notifikasi di LUAR transaksi DB, lewat job queue (§8.3) — kegagalan
	// enqueue TIDAK membatalkan approval yang sudah tersimpan.
	shortURL := s.frontendURL + "/" + key
	s.enqueueApprovedNotifications(ctx, req, shortURL)

	updated, ferr := s.repo.FindByID(ctx, id)
	if ferr != nil {
		log.Printf("[SHORTLINK-REQUEST] approveRequest: request %d disetujui tapi gagal re-fetch untuk response: %v", id, ferr)
		now := time.Now()
		req.Status = shortlinkrequest_model.StatusApproved
		req.ShortLinkID = &shortLinkID
		req.ShortKey = key
		req.ReviewedBy = reviewerID
		req.ReviewedVia = reviewedVia
		req.ReviewedDate = &now
		return s.toResponse(req), nil
	}
	return s.toResponse(updated), nil
}

// rejectRequest adalah SATU-SATUNYA jalan memproses reject — simetris dengan
// approveRequest, dipakai Reject (CMS) & HandleWhatsAppReply (WhatsApp).
func (s *ServiceImpl) rejectRequest(ctx context.Context, id int64, reviewerID *int64, reviewedVia string, reason *string) error {
	req, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return shortlinkrequest_repository.ErrNotFound
	}
	if req.Status != shortlinkrequest_model.StatusPending {
		return shortlinkrequest_repository.ErrAlreadyProcessed
	}
	if err := s.repo.UpdateStatus(ctx, id, shortlinkrequest_model.StatusRejected, reviewerID, reviewedVia, reason); err != nil {
		return err
	}
	s.enqueueRejectedNotifications(ctx, req, *reason)
	return nil
}

// Approve adalah pembungkus tipis approveRequest untuk jalur CMS (§7).
func (s *ServiceImpl) Approve(ctx context.Context, id, reviewerID int64) (shortlinkrequest_dto.Response, error) {
	rid := reviewerID
	res, err := s.approveRequest(ctx, id, &rid, shortlinkrequest_model.ReviewedViaCMS)
	switch {
	case errors.Is(err, shortlinkrequest_repository.ErrNotFound):
		return shortlinkrequest_dto.Response{}, apperror.NotFound("Permintaan tidak ditemukan")
	case errors.Is(err, shortlinkrequest_repository.ErrAlreadyProcessed):
		return shortlinkrequest_dto.Response{}, apperror.Conflict("Permintaan sudah diproses sebelumnya")
	case errors.Is(err, shortlinkrequest_repository.ErrKeyCollision):
		return shortlinkrequest_dto.Response{}, apperror.Conflict("Kunci shortlink sudah dipakai")
	case err != nil:
		return shortlinkrequest_dto.Response{}, apperror.Internal("Gagal menyetujui permintaan")
	}
	return res, nil
}

// Reject adalah pembungkus tipis rejectRequest untuk jalur CMS (§7).
func (s *ServiceImpl) Reject(ctx context.Context, id, reviewerID int64, reason string) error {
	rid := reviewerID
	err := s.rejectRequest(ctx, id, &rid, shortlinkrequest_model.ReviewedViaCMS, &reason)
	switch {
	case errors.Is(err, shortlinkrequest_repository.ErrNotFound):
		return apperror.NotFound("Permintaan tidak ditemukan")
	case errors.Is(err, shortlinkrequest_repository.ErrAlreadyProcessed):
		return apperror.Conflict("Permintaan sudah diproses sebelumnya")
	case err != nil:
		return apperror.Internal("")
	}
	return nil
}

// resolveTargetRequest menentukan shortLinkRequestID yang dimaksud sebuah
// balasan WhatsApp (§1a.5 techspec) — reply-threading (context.id) lebih
// dulu, baru fallback ke pencocokan pending-terbaru milik nomor yang sama.
// TIDAK PERNAH menebak kalau ada >1 kandidat ambigu.
func (s *ServiceImpl) resolveTargetRequest(ctx context.Context, payload kirimdev.InboundWebhookPayload) (int64, bool) {
	if payload.Context != nil && payload.Context.ID != "" {
		correlationType, correlationID, found, err := s.resolver.ResolveCorrelation(ctx, payload.Context.ID)
		if err == nil && found && correlationType == jobqueue_model.CorrelationTypeShortlinkRequest {
			if req, ferr := s.repo.FindByID(ctx, correlationID); ferr == nil && req.Status == shortlinkrequest_model.StatusPending {
				return correlationID, true
			}
		}
	}

	normalized := normalizePhone(payload.From)
	ids, err := s.resolver.FindRecentByPhone(ctx, normalized, jobqueue_model.CorrelationTypeShortlinkRequest, 20)
	if err != nil || len(ids) == 0 {
		return 0, false
	}
	pending, err := s.repo.FindPendingByIDs(ctx, ids)
	if err != nil || len(pending) != 1 {
		return 0, false // 0 atau >1 kandidat — TIDAK menebak (§1a.5)
	}
	return pending[0].ShortLinkRequestID, true
}

func (s *ServiceImpl) HandleWhatsAppReply(ctx context.Context, payload kirimdev.InboundWebhookPayload) (WhatsAppReplyOutcome, error) {
	picWhatsapp, _ := s.setting.GetValue(ctx, setting_model.GroupLayanan, setting_model.KeyShortlinkPICWhatsapp)
	if picWhatsapp == "" {
		return OutcomeIgnoredNotPIC, nil
	}
	// picWhatsapp datang dari field App Settings bebas format (cuma label
	// hint "format 62xxxxxxxxxx", tidak divalidasi ketat) — normalisasi juga
	// sebelum dibandingkan, supaya admin mengisi "+62..."/"08..." tidak
	// diam-diam mematikan seluruh fitur approval via WhatsApp.
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
		_, err := s.approveRequest(ctx, requestID, nil, shortlinkrequest_model.ReviewedViaWhatsApp)
		switch {
		case errors.Is(err, shortlinkrequest_repository.ErrAlreadyProcessed):
			return OutcomeAlreadyProcessed, nil
		case errors.Is(err, shortlinkrequest_repository.ErrKeyCollision):
			s.handleApproveCollision(requestID)
			return OutcomeCollisionManualReview, nil
		case err != nil:
			return OutcomeApproved, err
		}
		return OutcomeApproved, nil
	}

	reason := "Ditolak oleh PIC via WhatsApp (tanpa alasan tertulis)"
	err := s.rejectRequest(ctx, requestID, nil, shortlinkrequest_model.ReviewedViaWhatsApp, &reason)
	if errors.Is(err, shortlinkrequest_repository.ErrAlreadyProcessed) {
		return OutcomeAlreadyProcessed, nil
	}
	return OutcomeRejected, err
}

// handleApproveCollision menangani kunci yang diminta sudah dipakai sejak
// submission — TIDAK auto-create, request tetap 'pending' (ApproveTx sudah
// rollback bersih via ErrKeyCollision). Simplifikasi: cukup di-log untuk
// observability, TIDAK mengirim balasan WhatsApp — WhatsApp Business API
// mengharuskan pesan business-initiated pakai template pra-approved, dan
// tidak ada template "collision notice" terdaftar (§8.2 techspec cuma
// mendaftarkan 3 template inti). Admin tetap bisa lihat request ini masih
// 'pending' di antrian CMS dan memprosesnya manual.
func (s *ServiceImpl) handleApproveCollision(requestID int64) {
	log.Printf("[SHORTLINK-REQUEST] HandleWhatsAppReply: request %d collision saat approve via WhatsApp — perlu proses manual lewat CMS", requestID)
}

func (s *ServiceImpl) enqueueApprovedNotifications(ctx context.Context, req shortlinkrequest_model.ShortLinkRequest, shortURL string) {
	if _, err := s.jobs.Enqueue(ctx, jobqueue_dto.EnqueueInput{
		Queue: jobqueue_model.QueueWhatsApp, JobType: jobqueue_model.JobTypeWhatsAppTemplate,
		Payload: kirimdev.TemplateMessage{
			ToPhone: req.RequesterWhatsapp, TemplateName: "shortlink_approved",
			Params: []string{req.RequesterName, shortURL},
		},
		CorrelationType: jobqueue_model.CorrelationTypeShortlinkRequest, CorrelationID: req.ShortLinkRequestID,
	}); err != nil {
		log.Printf("[SHORTLINK-REQUEST] Approve: gagal enqueue WA ke requester: %v", err)
	}
	if _, err := s.jobs.Enqueue(ctx, jobqueue_dto.EnqueueInput{
		Queue: jobqueue_model.QueueEmail, JobType: jobqueue_model.JobTypeEmailShortlinkApproved,
		Payload:         jobqueue_dto.ShortlinkApprovedEmailPayload{ToEmail: req.RequesterEmail, ToName: req.RequesterName, ShortURL: shortURL},
		CorrelationType: jobqueue_model.CorrelationTypeShortlinkRequest, CorrelationID: req.ShortLinkRequestID,
	}); err != nil {
		log.Printf("[SHORTLINK-REQUEST] Approve: gagal enqueue email ke requester: %v", err)
	}
}

func (s *ServiceImpl) enqueueRejectedNotifications(ctx context.Context, req shortlinkrequest_model.ShortLinkRequest, reason string) {
	if _, err := s.jobs.Enqueue(ctx, jobqueue_dto.EnqueueInput{
		Queue: jobqueue_model.QueueWhatsApp, JobType: jobqueue_model.JobTypeWhatsAppTemplate,
		Payload: kirimdev.TemplateMessage{
			ToPhone: req.RequesterWhatsapp, TemplateName: "shortlink_rejected",
			Params: []string{req.RequesterName, reason},
		},
		CorrelationType: jobqueue_model.CorrelationTypeShortlinkRequest, CorrelationID: req.ShortLinkRequestID,
	}); err != nil {
		log.Printf("[SHORTLINK-REQUEST] Reject: gagal enqueue WA ke requester: %v", err)
	}
	if _, err := s.jobs.Enqueue(ctx, jobqueue_dto.EnqueueInput{
		Queue: jobqueue_model.QueueEmail, JobType: jobqueue_model.JobTypeEmailShortlinkRejected,
		Payload:         jobqueue_dto.ShortlinkRejectedEmailPayload{ToEmail: req.RequesterEmail, ToName: req.RequesterName, Reason: reason},
		CorrelationType: jobqueue_model.CorrelationTypeShortlinkRequest, CorrelationID: req.ShortLinkRequestID,
	}); err != nil {
		log.Printf("[SHORTLINK-REQUEST] Reject: gagal enqueue email ke requester: %v", err)
	}
}
