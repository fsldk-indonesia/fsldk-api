// Package qrcoderequest_repository adalah lapisan akses data modul QR Code
// request (GORM).
package qrcoderequest_repository

import (
	"context"
	"errors"

	"fsldk-api/modules/qrcode/qrcode_model"
	"fsldk-api/modules/qrcode/qrcoderequest_dto"
	"fsldk-api/modules/qrcode/qrcoderequest_model"
)

// ErrNotFound dikembalikan bila permintaan QR Code tidak ditemukan.
var ErrNotFound = errors.New("permintaan qr code tidak ditemukan")

// ErrAlreadyProcessed dikembalikan ApproveTx/UpdateStatus saat UPDATE
// kondisional (WHERE status='pending') tidak mengenai baris apa pun — berarti
// request sudah diproses jalur lain (CMS atau WhatsApp) tepat sebelum panggilan
// ini. Backstop otentik untuk race condition dua jalur approval.
var ErrAlreadyProcessed = errors.New("permintaan sudah diproses")

// Repository adalah kontrak akses data QR Code request.
type Repository interface {
	FindByID(ctx context.Context, id int64) (qrcoderequest_model.QRCodeRequest, error)
	// FindPendingByIDs mengembalikan subset dari ids yang statusnya masih
	// 'pending' — dipakai resolusi balasan WhatsApp saat fallback ke
	// pencocokan pending-terbaru.
	FindPendingByIDs(ctx context.Context, ids []int64) ([]qrcoderequest_model.QRCodeRequest, error)
	List(ctx context.Context, f qrcoderequest_dto.ListFilter) ([]qrcoderequest_model.QRCodeRequest, int64, error)
	Create(ctx context.Context, req qrcoderequest_dto.SubmitRequest) (int64, error)
	// ApproveTx menjalankan satu transaksi atomik: membuat baris ms_qrcode baru
	// dari nilai model `qr` (destinationURL + kustomisasi warna/ikon/caption yang
	// dipilih pemohon) + menandai permintaan sebagai approved KONDISIONAL (WHERE
	// status='pending'). reviewerID nil untuk penyelesaian via WhatsApp.
	ApproveTx(ctx context.Context, requestID int64, qr qrcode_model.QRCode, reviewerID *int64, reviewedVia string) (int64, error)
	// UpdateStatus dipakai untuk Reject — juga kondisional (WHERE status='pending').
	UpdateStatus(ctx context.Context, requestID int64, status string, reviewerID *int64, reviewedVia string, rejectionReason *string) error
}
