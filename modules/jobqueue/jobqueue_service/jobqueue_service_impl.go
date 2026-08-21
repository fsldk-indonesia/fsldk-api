package jobqueue_service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/config"
	"fsldk-api/modules/jobqueue/jobqueue_dto"
	"fsldk-api/modules/jobqueue/jobqueue_model"
	"fsldk-api/modules/jobqueue/jobqueue_repository"
	"fsldk-api/pkg/kirimdev"
	"fsldk-api/pkg/mailer"

	"golang.org/x/time/rate"
)

// sortColumns memetakan field sort yang diizinkan ke kolom database.
var sortColumns = map[string]string{
	"createdDate": "createdDate",
	"status":      "status",
	"queue":       "queue",
}

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct {
	repo     jobqueue_repository.Repository
	kirimdev *kirimdev.Client
	mailer   mailer.Mailer
	cfg      config.AppConfig
	limiters map[string]*rate.Limiter
}

// NewService membuat Service job queue.
func NewService(repo jobqueue_repository.Repository, kirimdevClient *kirimdev.Client, mail mailer.Mailer, cfg config.AppConfig) Service {
	// int(...) pada rate < 1/menit akan terpotong jadi burst=0, yang membuat
	// rate.Limiter.Allow() SELALU false (tidak pernah mengisi ulang) — WA
	// akan macet permanen kalau config di-set sub-1/menit. Minimal 1 supaya
	// limiter tetap fungsional pada rate serendah apa pun yang > 0.
	whatsappBurst := int(cfg.JobQueueWhatsAppRatePerMinute)
	if whatsappBurst < 1 {
		whatsappBurst = 1
	}
	return &ServiceImpl{
		repo: repo, kirimdev: kirimdevClient, mailer: mail, cfg: cfg,
		limiters: map[string]*rate.Limiter{
			// Inf: queue "email" tanpa batas — burst 0 tetap mengizinkan semua
			// event karena limiter Inf tidak pernah membatasi (lihat dok x/time/rate).
			jobqueue_model.QueueWhatsApp: rate.NewLimiter(rate.Limit(cfg.JobQueueWhatsAppRatePerMinute/60.0), whatsappBurst),
			jobqueue_model.QueueEmail:    rate.NewLimiter(rate.Inf, 0),
		},
	}
}

func (s *ServiceImpl) Enqueue(ctx context.Context, in jobqueue_dto.EnqueueInput) (int64, error) {
	payloadBytes, err := json.Marshal(in.Payload)
	if err != nil {
		return 0, err
	}

	maxAttempts := in.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = s.cfg.JobQueueDefaultMaxAttempts
	}

	available := time.Now().Add(time.Duration(in.DelaySeconds) * time.Second)
	if in.Queue == jobqueue_model.QueueWhatsApp {
		// Jitter 0-8 detik anti-burst, menyamai referensi (delay(rand(0,8)s)).
		available = available.Add(time.Duration(rand.Intn(8000)) * time.Millisecond)
	}

	job := jobqueue_model.Job{
		Queue: in.Queue, JobType: in.JobType, Payload: string(payloadBytes),
		MaxAttempts: maxAttempts, AvailableDate: available,
	}
	if in.CorrelationType != "" {
		correlationType := in.CorrelationType
		correlationID := in.CorrelationID
		job.CorrelationType = &correlationType
		job.CorrelationID = &correlationID
	}
	return s.repo.Create(ctx, job)
}

func (s *ServiceImpl) ResolveCorrelation(ctx context.Context, waMessageID string) (string, int64, bool, error) {
	return s.repo.ResolveCorrelation(ctx, waMessageID)
}

func (s *ServiceImpl) FindRecentByPhone(ctx context.Context, phone, correlationType string, limit int) ([]int64, error) {
	return s.repo.FindRecentByPhone(ctx, phone, correlationType, limit)
}

// deliveryStatusFailed adalah nilai "status" pada event message.status
// Kirimdev (§12 techspec) — domain berbeda dari jobqueue_model.StatusFailed
// (status job internal) meski kebetulan string-nya sama, sengaja dipisah
// biar tidak tercampur secara konseptual.
const deliveryStatusFailed = "failed"

func (s *ServiceImpl) HandleDeliveryStatus(ctx context.Context, waMessageID, status, errorDetail string) error {
	if status != deliveryStatusFailed {
		return nil
	}
	jobID, found, err := s.repo.FindJobIDByWAMessageID(ctx, waMessageID)
	if err != nil {
		return err
	}
	if !found {
		log.Printf("[JOBQUEUE] status pengiriman 'failed' untuk waMessageID %s tidak ketemu di tr_whatsapp_message_log — diabaikan", waMessageID)
		return nil
	}
	note := fmt.Sprintf("[Kirimdev] pengiriman gagal setelah initial accept (job tetap 'completed', ini cuma jejak): %s", errorDetail)
	log.Printf("[JOBQUEUE] job %d: %s (waMessageID=%s)", jobID, note, waMessageID)
	return s.repo.RecordDeliveryFailure(ctx, jobID, note)
}

func toResponse(j jobqueue_model.Job) jobqueue_dto.Response {
	lastError, correlationType := "", ""
	var correlationID int64
	if j.LastError != nil {
		lastError = *j.LastError
	}
	if j.CorrelationType != nil {
		correlationType = *j.CorrelationType
	}
	if j.CorrelationID != nil {
		correlationID = *j.CorrelationID
	}
	completedDate, failedDate := "", ""
	if j.CompletedDate != nil {
		completedDate = j.CompletedDate.Format("2006-01-02 15:04:05")
	}
	if j.FailedDate != nil {
		failedDate = j.FailedDate.Format("2006-01-02 15:04:05")
	}
	return jobqueue_dto.Response{
		JobID: j.JobID, Queue: j.Queue, JobType: j.JobType, Status: j.Status,
		Attempts: j.Attempts, MaxAttempts: j.MaxAttempts, LastError: lastError,
		CorrelationType: correlationType, CorrelationID: correlationID,
		AvailableDate: j.AvailableDate.Format("2006-01-02 15:04:05"),
		CreatedDate:   j.CreatedDate.Format("2006-01-02 15:04:05"),
		CompletedDate: completedDate, FailedDate: failedDate,
	}
}

func (s *ServiceImpl) CMSList(ctx context.Context, q dto.ListQuery, status, queue string) ([]jobqueue_dto.Response, int, error) {
	rows, total, err := s.repo.List(ctx, jobqueue_dto.ListFilter{
		Status: status, Queue: queue, Search: q.Search, Limit: q.Limit, Offset: q.Offset(),
		OrderBy: q.OrderBy(sortColumns, "createdDate DESC"),
	})
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	out := make([]jobqueue_dto.Response, 0, len(rows))
	for _, r := range rows {
		out = append(out, toResponse(r))
	}
	return out, int(total), nil
}

func (s *ServiceImpl) CMSGet(ctx context.Context, id int64) (jobqueue_dto.Response, error) {
	j, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return jobqueue_dto.Response{}, apperror.NotFound("Job tidak ditemukan")
	}
	return toResponse(j), nil
}

func (s *ServiceImpl) CMSStats(ctx context.Context) (jobqueue_dto.StatsResponse, error) {
	threshold := time.Duration(s.cfg.JobQueueStuckThresholdMinutes) * time.Minute
	stats, err := s.repo.Stats(ctx, threshold)
	if err != nil {
		return jobqueue_dto.StatsResponse{}, apperror.Internal("")
	}
	return stats, nil
}

func (s *ServiceImpl) Retry(ctx context.Context, id int64) error {
	err := s.repo.Retry(ctx, id)
	if errors.Is(err, jobqueue_repository.ErrInvalidState) {
		return apperror.Conflict("Job hanya bisa di-retry dari status failed")
	}
	if err != nil {
		return apperror.Internal("")
	}
	return nil
}

func (s *ServiceImpl) Delete(ctx context.Context, id int64) error {
	err := s.repo.Delete(ctx, id)
	if errors.Is(err, jobqueue_repository.ErrInvalidState) {
		return apperror.Conflict("Job hanya bisa dihapus dari status failed/completed")
	}
	if err != nil {
		return apperror.Internal("")
	}
	return nil
}

// RunWorker adalah blocking poll loop persisten — BUKAN worker bergilir
// lewat cron seperti referensi Laravel (§8.3 techspec): fsldk-api sudah
// berupa proses long-running, jadi polling sederhana sudah cukup.
func (s *ServiceImpl) RunWorker(workerID int) {
	interval := time.Duration(s.cfg.JobQueuePollIntervalMS) * time.Millisecond
	if interval <= 0 {
		interval = 1500 * time.Millisecond
	}
	queues := []string{jobqueue_model.QueueWhatsApp, jobqueue_model.QueueEmail}

	for {
		claimedAny := false
		for _, q := range queues {
			if lim, ok := s.limiters[q]; ok && !lim.Allow() {
				continue // rate-limited — dilewati tick ini, BUKAN kegagalan job
			}
			job, ok, err := s.repo.Claim(context.Background(), q)
			if err != nil {
				log.Printf("[JOBQUEUE] worker %d: gagal claim queue %s: %v", workerID, q, err)
				continue
			}
			if !ok {
				continue
			}
			claimedAny = true
			s.executeJob(context.Background(), job)
		}
		if !claimedAny {
			time.Sleep(interval)
		}
	}
}

func (s *ServiceImpl) executeJob(ctx context.Context, job jobqueue_model.Job) {
	var err error
	switch job.JobType {
	case jobqueue_model.JobTypeWhatsAppTemplate:
		err = s.executeWhatsAppTemplate(ctx, job)
	case jobqueue_model.JobTypeEmailShortlinkApproved:
		var p jobqueue_dto.ShortlinkApprovedEmailPayload
		if uerr := json.Unmarshal([]byte(job.Payload), &p); uerr != nil {
			err = uerr
		} else {
			err = s.mailer.SendShortlinkApprovedEmail(p.ToEmail, p.ToName, p.ShortURL)
		}
	case jobqueue_model.JobTypeEmailShortlinkRejected:
		var p jobqueue_dto.ShortlinkRejectedEmailPayload
		if uerr := json.Unmarshal([]byte(job.Payload), &p); uerr != nil {
			err = uerr
		} else {
			err = s.mailer.SendShortlinkRejectedEmail(p.ToEmail, p.ToName, p.Reason)
		}
	default:
		err = fmt.Errorf("jobqueue: jobType tidak dikenal %q", job.JobType)
	}

	if err != nil {
		s.handleJobFailure(ctx, job, err)
		return
	}
	if merr := s.repo.MarkCompleted(ctx, job.JobID); merr != nil {
		log.Printf("[JOBQUEUE] job %d selesai tapi gagal MarkCompleted: %v", job.JobID, merr)
	}
}

func (s *ServiceImpl) executeWhatsAppTemplate(ctx context.Context, job jobqueue_model.Job) error {
	var msg kirimdev.TemplateMessage
	if err := json.Unmarshal([]byte(job.Payload), &msg); err != nil {
		return err
	}
	result, err := s.kirimdev.SendTemplate(ctx, msg)
	if err != nil {
		return err
	}
	correlationType := ""
	var correlationID int64
	if job.CorrelationType != nil {
		correlationType = *job.CorrelationType
	}
	if job.CorrelationID != nil {
		correlationID = *job.CorrelationID
	}
	if logErr := s.repo.LogOutboundMessage(ctx, job.JobID, result.MessageID, msg.ToPhone, msg.TemplateName, correlationType, correlationID); logErr != nil {
		log.Printf("[JOBQUEUE] job %d: gagal catat tr_whatsapp_message_log: %v", job.JobID, logErr)
	}
	return nil
}

func (s *ServiceImpl) handleJobFailure(ctx context.Context, job jobqueue_model.Job, cause error) {
	if job.Attempts >= job.MaxAttempts {
		if merr := s.repo.MarkRetryOrFail(ctx, job.JobID, cause.Error(), nil); merr != nil {
			log.Printf("[JOBQUEUE] job %d: gagal MarkRetryOrFail(failed): %v", job.JobID, merr)
		}
		return
	}
	backoff := s.cfg.JobQueueBackoffSchedule()
	var wait time.Duration
	switch {
	case len(backoff) == 0:
		wait = 30 * time.Second
	case job.Attempts-1 < len(backoff):
		wait = backoff[job.Attempts-1]
	default:
		wait = backoff[len(backoff)-1]
	}
	next := time.Now().Add(wait)
	if merr := s.repo.MarkRetryOrFail(ctx, job.JobID, cause.Error(), &next); merr != nil {
		log.Printf("[JOBQUEUE] job %d: gagal MarkRetryOrFail(retry): %v", job.JobID, merr)
	}
}

// RunStuckSweeper me-recycle job yang macet di 'processing' — pengganti
// crash-recovery karena worker goroutine tidak punya graceful shutdown
// (§12 techspec).
func (s *ServiceImpl) RunStuckSweeper() {
	threshold := time.Duration(s.cfg.JobQueueStuckThresholdMinutes) * time.Minute
	if threshold <= 0 {
		threshold = 10 * time.Minute
	}
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		recycled, failed, err := s.repo.SweepStuck(context.Background(), threshold)
		if err != nil {
			log.Printf("[JOBQUEUE] sweeper: gagal sweep stuck jobs: %v", err)
			continue
		}
		if recycled > 0 || failed > 0 {
			log.Printf("[JOBQUEUE] sweeper: %d job di-recycle, %d job di-failed (stuck)", recycled, failed)
		}
	}
}
