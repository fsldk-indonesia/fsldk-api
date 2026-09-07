// Package qrcoderequest_service memuat logika bisnis modul QR Code request
// (alur permintaan publik + persetujuan admin di atas modul QR Code).
package qrcoderequest_service

import (
	"context"

	"fsldk-api/base/dto"
	"fsldk-api/modules/qrcode/qrcoderequest_dto"
	"fsldk-api/pkg/kirimdev"
)

// WhatsAppReplyOutcome menjelaskan apa yang terjadi terhadap satu balasan
// WhatsApp inbound — dipakai logging observability di handler webhook.
type WhatsAppReplyOutcome string

const (
	OutcomeApproved            WhatsAppReplyOutcome = "approved"
	OutcomeRejected            WhatsAppReplyOutcome = "rejected"
	OutcomeAlreadyProcessed    WhatsAppReplyOutcome = "already_processed"
	OutcomeIgnoredNotPIC       WhatsAppReplyOutcome = "ignored_not_pic"
	OutcomeIgnoredNoIntent     WhatsAppReplyOutcome = "ignored_no_intent"
	OutcomeAmbiguousOrNotFound WhatsAppReplyOutcome = "ambiguous_or_not_found"
)

// Service adalah kontrak logika bisnis QR Code request.
type Service interface {
	// Submit menyimpan permintaan baru (status default 'pending') dan mengantre
	// notifikasi WhatsApp ke PIC (best-effort, lewat job queue).
	Submit(ctx context.Context, req qrcoderequest_dto.SubmitRequest) (qrcoderequest_dto.Response, error)
	// PublicPIC mengembalikan info kontak PIC untuk kartu "Konfirmasi via
	// WhatsApp" di halaman publik pengajuan.
	PublicPIC(ctx context.Context) (qrcoderequest_dto.PICResponse, error)
	CMSList(ctx context.Context, q dto.ListQuery, status string) ([]qrcoderequest_dto.Response, int, error)
	CMSGet(ctx context.Context, id int64) (qrcoderequest_dto.Response, error)
	// Approve (jalur CMS) membuat QR Code baru (transaksi atomik kondisional)
	// dan mengantre notifikasi ke requester.
	Approve(ctx context.Context, id, reviewerID int64) (qrcoderequest_dto.Response, error)
	// Reject (jalur CMS) menandai permintaan ditolak dan mengantre notifikasi.
	Reject(ctx context.Context, id, reviewerID int64, reason string) error
	// HandleWhatsAppReply adalah jalur approval kedua — dipanggil handler
	// webhook Kirimdev setelah signature terverifikasi. Melewati mekanisme
	// atomik yang SAMA dengan Approve/Reject.
	HandleWhatsAppReply(ctx context.Context, payload kirimdev.InboundWebhookPayload) (WhatsAppReplyOutcome, error)
}
