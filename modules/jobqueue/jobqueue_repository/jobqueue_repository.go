// Package jobqueue_repository adalah lapisan akses data modul job queue (GORM).
package jobqueue_repository

import (
	"context"
	"errors"
	"time"

	"fsldk-api/modules/jobqueue/jobqueue_dto"
	"fsldk-api/modules/jobqueue/jobqueue_model"
)

// ErrNotFound dikembalikan bila job tidak ditemukan.
var ErrNotFound = errors.New("job tidak ditemukan")

// ErrInvalidState dikembalikan bila Retry/Delete dipanggil pada job yang
// statusnya tidak memenuhi syarat (mis. Retry pada job yang bukan 'failed').
var ErrInvalidState = errors.New("job tidak dalam status yang bisa diproses")

// Repository adalah kontrak akses data job queue.
type Repository interface {
	Create(ctx context.Context, j jobqueue_model.Job) (int64, error)
	FindByID(ctx context.Context, id int64) (jobqueue_model.Job, error)
	List(ctx context.Context, f jobqueue_dto.ListFilter) ([]jobqueue_model.Job, int64, error)
	Stats(ctx context.Context, stuckThreshold time.Duration) (jobqueue_dto.StatsResponse, error)

	// Claim mengambil 1 job pending teratas untuk queue tertentu secara
	// atomik (UPDATE ... WHERE status='pending' ...) — aman dipakai banyak
	// worker/instance sekaligus tanpa SELECT ... FOR UPDATE.
	Claim(ctx context.Context, queue string) (jobqueue_model.Job, bool, error)
	MarkCompleted(ctx context.Context, id int64) error
	// MarkRetryOrFail: nextAvailable=nil berarti job langsung 'failed'
	// (attempts sudah mencapai maxAttempts), selain itu 'pending' lagi
	// dengan availableDate=nextAvailable (backoff).
	MarkRetryOrFail(ctx context.Context, id int64, lastErr string, nextAvailable *time.Time) error
	// SweepStuck me-recycle job 'processing' yang macet melewati threshold
	// balik ke 'pending' (kalau masih ada sisa attempt) atau langsung 'failed'.
	SweepStuck(ctx context.Context, threshold time.Duration) (recycled, failed int64, err error)

	Retry(ctx context.Context, id int64) error  // hanya dari status='failed'
	Delete(ctx context.Context, id int64) error // hanya dari status IN ('failed','completed')

	LogOutboundMessage(ctx context.Context, jobID int64, waMessageID, toPhone, templateName, correlationType string, correlationID int64) error
	ResolveCorrelation(ctx context.Context, waMessageID string) (correlationType string, correlationID int64, found bool, err error)
	FindRecentByPhone(ctx context.Context, phone, correlationType string, limit int) ([]int64, error)

	// FindJobIDByWAMessageID & RecordDeliveryFailure dipakai menangani event
	// webhook "message.status" Kirimdev (§12 techspec) — job sudah
	// 'completed' duluan (API kirim menerima request dengan 200), kegagalan
	// pengiriman sesungguhnya (mis. template belum ada) baru diketahui
	// belakangan lewat webhook async, TIDAK mengubah status/attempts job,
	// cuma menandai lastError supaya kelihatan di CMS Job Queue.
	FindJobIDByWAMessageID(ctx context.Context, waMessageID string) (int64, bool, error)
	RecordDeliveryFailure(ctx context.Context, jobID int64, note string) error
}
