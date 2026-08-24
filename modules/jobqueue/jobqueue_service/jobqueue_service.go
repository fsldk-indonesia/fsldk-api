// Package jobqueue_service memuat logika bisnis modul job queue.
package jobqueue_service

import (
	"context"

	"fsldk-api/base/dto"
	"fsldk-api/modules/jobqueue/jobqueue_dto"
)

// Service adalah kontrak logika bisnis job queue.
type Service interface {
	// Enqueue dipakai modul lain (mis. shortlinkrequest_service) untuk
	// memasukkan job baru.
	Enqueue(ctx context.Context, in jobqueue_dto.EnqueueInput) (int64, error)
	// ResolveCorrelation & FindRecentByPhone dipakai resolusi balasan
	// WhatsApp (§1a.5 techspec) — WhatsAppMessageResolver di shortlinkrequest_service.
	ResolveCorrelation(ctx context.Context, waMessageID string) (correlationType string, correlationID int64, found bool, err error)
	FindRecentByPhone(ctx context.Context, phone, correlationType string, limit int) ([]int64, error)
	// HandleDeliveryStatus memproses event webhook "message.status" Kirimdev
	// (§12 techspec) — satu-satunya cara kegagalan pengiriman async (mis.
	// template belum ada, baru diketahui setelah API kirim awalnya menerima
	// request dengan 200) jadi kelihatan alih-alih hilang diam-diam. Status
	// selain "failed" diabaikan.
	HandleDeliveryStatus(ctx context.Context, waMessageID, status, errorDetail string) error
	// HandleMessageSent memproses event webhook "message.sent" Kirimdev —
	// respons sinkron SendTemplate cuma punya ID internal Kirimdev, wamid
	// ASLI baru muncul di event async ini (§1a.5/§1b techspec). Dipakai
	// memperbarui tr_whatsapp_message_log supaya context.id balasan PIC
	// (yang selalu wamid asli) bisa dicocokkan.
	HandleMessageSent(ctx context.Context, kirimdevMessageID, wamid string) error

	CMSList(ctx context.Context, q dto.ListQuery, status, queue string) ([]jobqueue_dto.Response, int, error)
	CMSGet(ctx context.Context, id int64) (jobqueue_dto.Response, error)
	CMSStats(ctx context.Context) (jobqueue_dto.StatsResponse, error)
	Retry(ctx context.Context, id int64) error
	Delete(ctx context.Context, id int64) error

	// RunWorker adalah blocking poll loop — dipanggil `go svc.RunWorker(i)`
	// sebanyak JOBQUEUE_WORKER_COUNT dari root router.go.
	RunWorker(workerID int)
	// RunStuckSweeper adalah blocking sweep loop — dipanggil sekali
	// `go svc.RunStuckSweeper()` dari root router.go.
	RunStuckSweeper()
}
