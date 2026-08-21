// Package shortlinkrequest_repository adalah lapisan akses data modul
// shortlink request (GORM).
package shortlinkrequest_repository

import (
	"context"
	"errors"

	"fsldk-api/modules/shortlink/shortlinkrequest_dto"
	"fsldk-api/modules/shortlink/shortlinkrequest_model"
)

// ErrNotFound dikembalikan bila permintaan shortlink tidak ditemukan.
var ErrNotFound = errors.New("permintaan shortlink tidak ditemukan")

// ErrAlreadyProcessed dikembalikan ApproveTx/UpdateStatus saat UPDATE
// kondisional (WHERE status='pending') tidak mengenai baris apa pun — berarti
// request sudah diproses jalur lain (CMS atau WhatsApp, §1a.5) tepat sebelum
// panggilan ini. Ini backstop otentik untuk race condition dua jalur approval.
var ErrAlreadyProcessed = errors.New("permintaan sudah diproses")

// ErrKeyCollision dikembalikan ApproveTx saat insert ms_shortlink kena
// duplicate-key pada UNIQUE(shortKey) — dua request/aktor berebut kunci yang
// sama nyaris bersamaan.
var ErrKeyCollision = errors.New("kunci shortlink sudah dipakai")

// Repository adalah kontrak akses data shortlink request.
type Repository interface {
	FindByID(ctx context.Context, id int64) (shortlinkrequest_model.ShortLinkRequest, error)
	// FindPendingByIDs mengembalikan subset dari ids yang statusnya masih
	// 'pending' — dipakai resolusi balasan WhatsApp saat fallback ke
	// pencocokan pending-terbaru (§1a.5 techspec).
	FindPendingByIDs(ctx context.Context, ids []int64) ([]shortlinkrequest_model.ShortLinkRequest, error)
	List(ctx context.Context, f shortlinkrequest_dto.ListFilter) ([]shortlinkrequest_model.ShortLinkRequest, int64, error)
	Create(ctx context.Context, req shortlinkrequest_dto.SubmitRequest) (int64, error)
	// ApproveTx menjalankan satu transaksi atomik: membuat baris ms_shortlink
	// baru + menandai permintaan sebagai approved — KONDISIONAL (WHERE
	// status='pending'), mengembalikan ErrAlreadyProcessed/ErrKeyCollision
	// kalau kalah race (§1a.5/§6 techspec). reviewerID nil untuk penyelesaian
	// via WhatsApp (reviewedVia="whatsapp").
	ApproveTx(ctx context.Context, requestID int64, destinationURL, shortKey string, reviewerID *int64, reviewedVia string) (int64, error)
	// UpdateStatus dipakai untuk Reject — juga kondisional (WHERE status='pending').
	UpdateStatus(ctx context.Context, requestID int64, status string, reviewerID *int64, reviewedVia string, rejectionReason *string) error
}
