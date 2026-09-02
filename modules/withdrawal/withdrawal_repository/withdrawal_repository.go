// Package withdrawal_repository adalah lapisan akses data modul withdrawal (GORM).
package withdrawal_repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"fsldk-api/modules/withdrawal/withdrawal_dto"
	"fsldk-api/modules/withdrawal/withdrawal_model"
)

// ErrNotFound dikembalikan bila withdrawal tidak ditemukan.
var ErrNotFound = errors.New("withdrawal tidak ditemukan")

// ErrDuplicateIdempotencyKey dikembalikan bila insert ditolak oleh
// UNIQUE(idempotencyKey).
var ErrDuplicateIdempotencyKey = errors.New("idempotencyKey sudah pernah dipakai")

// Repository adalah kontrak akses data withdrawal.
type Repository interface {
	// Create menyisipkan baris withdrawal baru di dalam tx yang diberikan —
	// wajib satu transaksi dengan wallet_service.ReserveWithdrawal (caller-
	// controlled, lihat withdrawal_service.Request).
	Create(tx *gorm.DB, p withdrawal_model.CreateParams) (int64, error)
	FindByID(ctx context.Context, id int64) (withdrawal_model.Withdrawal, error)
	FindByIdempotencyKey(ctx context.Context, key string) (withdrawal_model.Withdrawal, error)
	// CountNonFinalByCampaign menghitung withdrawal campaign yang masih
	// berjalan (REQUESTED..PROCESSING) — pre-check cepat sebelum memanggil
	// inquiry gateway (fail-fast, tidak concurrency-safe sendirian).
	CountNonFinalByCampaign(ctx context.Context, campaignID int64) (int64, error)
	// CountNonFinalByCampaignForUpdate adalah varian locking read (dalam tx)
	// yang jadi penjamin sesungguhnya "hanya satu withdrawal aktif per
	// campaign" — wajib dipanggil di dalam tx yang sama dengan Create,
	// setelah baris ms_campaign terkunci (lihat withdrawal_service.Request).
	CountNonFinalByCampaignForUpdate(tx *gorm.DB, campaignID int64) (int64, error)
	List(ctx context.Context, f withdrawal_dto.ListFilter) ([]withdrawal_model.Withdrawal, int64, error)

	// UpdateStatus memperbarui status withdrawal di dalam tx yang diberikan
	// — dipakai seluruh transisi (approve/reject/process/cancel/callback).
	UpdateStatus(tx *gorm.DB, id int64, status string, p withdrawal_model.StatusUpdateParams) error
	// FindByRefForUpdate mengunci baris withdrawal (SELECT...FOR UPDATE) di
	// dalam tx — wajib dipanggil sebelum UpdateStatus dari callback disbursement.
	FindByRefForUpdate(tx *gorm.DB, withdrawalRef string) (withdrawal_model.Withdrawal, error)
	// FindStaleProcessing mengembalikan withdrawal berstatus PROCESSING yang
	// sudah melewati threshold waktu — kandidat rekonsiliasi manual.
	FindStaleProcessing(ctx context.Context, olderThan time.Time) ([]withdrawal_model.Withdrawal, error)
	// CountSuccessByCampaign menghitung withdrawal SUCCESS milik campaign —
	// dipakai trigger risk-based "withdrawal pertama campaign tsb" (§12.1).
	CountSuccessByCampaign(ctx context.Context, campaignID int64) (int64, error)

	// CreateOtpChallenge menyisipkan OTP challenge baru.
	CreateOtpChallenge(ctx context.Context, p withdrawal_model.OtpChallengeParams) (int64, error)
	// FindActiveOtpChallenge mengembalikan challenge terbaru milik withdrawal
	// yang belum diverifikasi dan belum kedaluwarsa.
	FindActiveOtpChallenge(ctx context.Context, withdrawalID int64) (withdrawal_model.OtpChallenge, error)
	// IncrementOtpAttempt menaikkan attemptCount — dipanggil setiap percobaan
	// verifikasi (benar maupun salah), penjamin rate limit §12.9.
	IncrementOtpAttempt(ctx context.Context, challengeID int64) error
	// MarkOtpVerified menandai challenge terverifikasi (sekali pakai).
	MarkOtpVerified(ctx context.Context, challengeID int64) error
	// CountOtpChallengesByWithdrawal menghitung total challenge (baru maupun
	// kedaluwarsa/gagal) yang pernah dibuat untuk satu withdrawal — dipakai
	// membatasi jumlah kumulatif permintaan OTP baru per withdrawal (Phase 13
	// security hardening), bukan hanya percobaan per challenge.
	CountOtpChallengesByWithdrawal(ctx context.Context, withdrawalID int64) (int64, error)
}
