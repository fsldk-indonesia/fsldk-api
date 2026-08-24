// Package shortlinkrequest_service memuat logika bisnis modul shortlink
// request (alur permintaan publik + persetujuan admin di atas modul shortlink).
package shortlinkrequest_service

import (
	"context"

	"fsldk-api/base/dto"
	"fsldk-api/modules/shortlink/shortlinkrequest_dto"
	"fsldk-api/pkg/kirimdev"
)

// WhatsAppReplyOutcome menjelaskan apa yang terjadi terhadap satu balasan
// WhatsApp inbound — dipakai logging observability di handler (§7 techspec).
type WhatsAppReplyOutcome string

const (
	OutcomeApproved              WhatsAppReplyOutcome = "approved"
	OutcomeRejected              WhatsAppReplyOutcome = "rejected"
	OutcomeAlreadyProcessed      WhatsAppReplyOutcome = "already_processed"
	OutcomeCollisionManualReview WhatsAppReplyOutcome = "collision_manual_review_needed"
	OutcomeIgnoredNotPIC         WhatsAppReplyOutcome = "ignored_not_pic"
	OutcomeIgnoredNoIntent       WhatsAppReplyOutcome = "ignored_no_intent"
	OutcomeAmbiguousOrNotFound   WhatsAppReplyOutcome = "ambiguous_or_not_found"
)

// Service adalah kontrak logika bisnis shortlink request.
type Service interface {
	// Submit menyimpan permintaan baru (status default 'pending') dan
	// mengantre notifikasi WhatsApp ke PIC (best-effort, lewat job queue).
	Submit(ctx context.Context, req shortlinkrequest_dto.SubmitRequest) (shortlinkrequest_dto.Response, error)
	// PublicPIC mengembalikan info kontak PIC untuk kartu "Konfirmasi via
	// WhatsApp" di halaman publik pengajuan.
	PublicPIC(ctx context.Context) (shortlinkrequest_dto.PICResponse, error)
	CMSList(ctx context.Context, q dto.ListQuery, status string) ([]shortlinkrequest_dto.Response, int, error)
	CMSGet(ctx context.Context, id int64) (shortlinkrequest_dto.Response, error)
	// Approve (jalur CMS) membuat shortlink baru (transaksi atomik kondisional,
	// §6) dan mengantre notifikasi ke requester.
	Approve(ctx context.Context, id, reviewerID int64) (shortlinkrequest_dto.Response, error)
	// Reject (jalur CMS) menandai permintaan ditolak dan mengantre notifikasi
	// ke requester.
	Reject(ctx context.Context, id, reviewerID int64, reason string) error
	// HandleWhatsAppReply adalah jalur approval kedua (§1a.5 techspec) —
	// dipanggil handler webhook Kirimdev setelah signature terverifikasi.
	// Melewati mekanisme atomik yang SAMA dengan Approve/Reject (§6), jadi
	// race antara jalur CMS & WhatsApp pada request yang sama selalu aman.
	HandleWhatsAppReply(ctx context.Context, payload kirimdev.InboundWebhookPayload) (WhatsAppReplyOutcome, error)
}
