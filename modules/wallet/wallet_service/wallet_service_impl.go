package wallet_service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"gorm.io/gorm"

	"fsldk-api/base/apperror"
	"fsldk-api/constants"
	"fsldk-api/modules/campaign/campaign_repository"
	"fsldk-api/modules/wallet/wallet_dto"
	"fsldk-api/modules/wallet/wallet_model"
	"fsldk-api/modules/wallet/wallet_repository"
)

type ServiceImpl struct {
	repo         wallet_repository.Repository
	campaignRepo campaign_repository.Repository
	db           *gorm.DB // hanya dipakai AdjustBalance, yang membuka transaksinya sendiri
}

func NewService(repo wallet_repository.Repository, campaignRepo campaign_repository.Repository, db *gorm.DB) Service {
	return &ServiceImpl{repo: repo, campaignRepo: campaignRepo, db: db}
}

func (s *ServiceImpl) CreditDonation(tx *gorm.DB, campaignID, donationID int64, amount float64, note string) error {
	_, _, err := s.repo.InsertLedgerEntry(tx, wallet_model.LedgerEntryParams{
		CampaignID:    campaignID,
		EntryType:     constants.LedgerEntryDonationCredit,
		Direction:     constants.LedgerDirectionCredit,
		Amount:        amount,
		ReferenceType: constants.LedgerReferenceDonation,
		ReferenceID:   donationID,
		Note:          nullString(note),
	})
	return ignoreDuplicateLedgerEntry(err)
}

func (s *ServiceImpl) ReserveWithdrawal(tx *gorm.DB, campaignID, withdrawalID int64, amount float64, actorUserID int64, note string) error {
	// Kunci baris campaign & baca saldo terkini sebelum divalidasi —
	// mencegah dua request withdrawal bersamaan sama-sama lolos validasi
	// padahal gabungan keduanya melebihi saldo. InsertLedgerEntry di bawah
	// akan mengunci ulang baris yang sama (reentrant, aman) sebelum
	// menghitung balanceAfter final.
	available, err := s.repo.LatestBalanceForUpdate(tx, campaignID)
	if err != nil {
		return err
	}
	if amount > available {
		return apperror.InsufficientBalance("")
	}

	_, _, err = s.repo.InsertLedgerEntry(tx, wallet_model.LedgerEntryParams{
		CampaignID:      campaignID,
		EntryType:       constants.LedgerEntryWithdrawalReserve,
		Direction:       constants.LedgerDirectionDebit,
		Amount:          amount,
		ReferenceType:   constants.LedgerReferenceWithdrawal,
		ReferenceID:     withdrawalID,
		Note:            nullString(note),
		CreatedByUserID: nullInt64(actorUserID),
	})
	return ignoreDuplicateLedgerEntry(err)
}

func (s *ServiceImpl) ReleaseWithdrawal(tx *gorm.DB, campaignID, withdrawalID int64, amount float64, note string) error {
	_, _, err := s.repo.InsertLedgerEntry(tx, wallet_model.LedgerEntryParams{
		CampaignID:    campaignID,
		EntryType:     constants.LedgerEntryWithdrawalRelease,
		Direction:     constants.LedgerDirectionCredit,
		Amount:        amount,
		ReferenceType: constants.LedgerReferenceWithdrawal,
		ReferenceID:   withdrawalID,
		Note:          nullString(note),
	})
	return ignoreDuplicateLedgerEntry(err)
}

func (s *ServiceImpl) RefundDebit(tx *gorm.DB, campaignID, donationID int64, amount float64, actorUserID int64, note string) error {
	// OQ-21: refund yang melebihi availableBalance ditolak sistem (tidak
	// ada dana talangan platform), dieskalasi manual ke admin/finance.
	available, err := s.repo.LatestBalanceForUpdate(tx, campaignID)
	if err != nil {
		return err
	}
	if amount > available {
		return apperror.InsufficientBalance("Saldo campaign tidak mencukupi untuk refund ini — perlu penyelesaian manual")
	}

	_, _, err = s.repo.InsertLedgerEntry(tx, wallet_model.LedgerEntryParams{
		CampaignID:      campaignID,
		EntryType:       constants.LedgerEntryRefundDebit,
		Direction:       constants.LedgerDirectionDebit,
		Amount:          amount,
		ReferenceType:   constants.LedgerReferenceDonation,
		ReferenceID:     donationID,
		Note:            nullString(note),
		CreatedByUserID: nullInt64(actorUserID),
	})
	return ignoreDuplicateLedgerEntry(err)
}

func (s *ServiceImpl) AdjustBalance(ctx context.Context, campaignID int64, amount float64, direction string, actorUserID int64, reason string) error {
	if reason == "" {
		return apperror.BadRequest("Alasan adjustment wajib diisi")
	}
	if direction != constants.LedgerDirectionCredit && direction != constants.LedgerDirectionDebit {
		return apperror.BadRequest("Arah adjustment tidak valid")
	}

	entryType := constants.LedgerEntryAdjustmentCredit
	if direction == constants.LedgerDirectionDebit {
		entryType = constants.LedgerEntryAdjustmentDebit
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if direction == constants.LedgerDirectionDebit {
			available, err := s.repo.LatestBalanceForUpdate(tx, campaignID)
			if err != nil {
				return err
			}
			if amount > available {
				return apperror.InsufficientBalance("")
			}
		}

		_, _, err := s.repo.InsertLedgerEntry(tx, wallet_model.LedgerEntryParams{
			CampaignID:    campaignID,
			EntryType:     entryType,
			Direction:     direction,
			Amount:        amount,
			ReferenceType: constants.LedgerReferenceAdjustment,
			// Adjustment tidak mengacu ke baris entity lain (bukan donasi/
			// withdrawal), tapi uq_ledger_reference(referenceType,
			// referenceID, entryType) tetap berlaku global — referenceID
			// harus unik per panggilan, bukan konstan. Nanosecond clock
			// cukup untuk Phase 1 (AdjustBalance dijalankan di dalam tx
			// dengan baris campaign terkunci, sehingga single-writer per
			// campaign per saat). Saat endpoint CMS wallet.adjust dibangun
			// (Phase 6+), pertimbangkan mengganti ini dengan idempotencyKey
			// eksplisit dari request admin.
			ReferenceID:     time.Now().UnixNano(),
			Note:            nullString(reason),
			CreatedByUserID: nullInt64(actorUserID),
		})
		return err
	})
}

func (s *ServiceImpl) GetBalance(ctx context.Context, campaignID int64) (wallet_dto.BalanceResponse, error) {
	available, err := s.repo.LatestBalance(ctx, campaignID)
	if err != nil {
		return wallet_dto.BalanceResponse{}, apperror.Internal("")
	}
	pending, err := s.repo.SumPendingReserve(ctx, campaignID)
	if err != nil {
		return wallet_dto.BalanceResponse{}, apperror.Internal("")
	}
	totalCollected, err := s.repo.SumByEntryType(ctx, campaignID, constants.LedgerEntryDonationCredit)
	if err != nil {
		return wallet_dto.BalanceResponse{}, apperror.Internal("")
	}
	totalWithdrawn, err := s.repo.SumWithdrawnSuccess(ctx, campaignID)
	if err != nil {
		return wallet_dto.BalanceResponse{}, apperror.Internal("")
	}

	return wallet_dto.BalanceResponse{
		AvailableBalance: available,
		PendingBalance:   pending,
		TotalCollected:   totalCollected,
		TotalWithdrawn:   totalWithdrawn,
	}, nil
}

func (s *ServiceImpl) ListLedger(ctx context.Context, campaignID int64, filter wallet_dto.LedgerListFilter) ([]wallet_dto.LedgerListItem, int64, error) {
	entries, total, err := s.repo.ListLedger(ctx, campaignID, filter)
	if err != nil {
		return nil, 0, apperror.Internal("")
	}

	items := make([]wallet_dto.LedgerListItem, 0, len(entries))
	for _, e := range entries {
		items = append(items, wallet_dto.LedgerListItem{
			LedgerID:      e.LedgerID,
			EntryType:     e.EntryType,
			Direction:     e.Direction,
			Amount:        e.Amount,
			BalanceAfter:  e.BalanceAfter,
			ReferenceType: e.ReferenceType,
			ReferenceID:   e.ReferenceID,
			Note:          e.Note.String,
			CreatedDate:   e.CreatedDate,
		})
	}
	return items, total, nil
}

func (s *ServiceImpl) GetBalanceForOwner(ctx context.Context, campaignID, ownerUserID int64) (wallet_dto.BalanceResponse, error) {
	if err := s.checkOwnership(ctx, campaignID, ownerUserID); err != nil {
		return wallet_dto.BalanceResponse{}, err
	}
	return s.GetBalance(ctx, campaignID)
}

func (s *ServiceImpl) ListLedgerForOwner(ctx context.Context, campaignID, ownerUserID int64, filter wallet_dto.LedgerListFilter) ([]wallet_dto.LedgerListItem, int64, error) {
	if err := s.checkOwnership(ctx, campaignID, ownerUserID); err != nil {
		return nil, 0, err
	}
	return s.ListLedger(ctx, campaignID, filter)
}

// checkOwnership menolak akses dengan 404 (bukan 403) bila campaign bukan
// milik ownerUserID — lihat komentar GetBalanceForOwner.
func (s *ServiceImpl) checkOwnership(ctx context.Context, campaignID, ownerUserID int64) error {
	camp, err := s.campaignRepo.FindByID(ctx, campaignID)
	if err != nil || camp.OwnerUserID != ownerUserID {
		return apperror.NotFound("Campaign tidak ditemukan")
	}
	return nil
}

// ignoreDuplicateLedgerEntry menerapkan idempotency ledger di level
// service: jika event yang sama sudah pernah dicatat (guard UNIQUE di DB,
// lihat wallet_repository.ErrDuplicateLedgerEntry), ini bukan kegagalan —
// caller (donation callback, dst) harus ack sebagai sukses tanpa reprocess.
func ignoreDuplicateLedgerEntry(err error) error {
	if err == nil || errors.Is(err, wallet_repository.ErrDuplicateLedgerEntry) {
		return nil
	}
	return err
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullInt64(id int64) sql.NullInt64 {
	if id == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: id, Valid: true}
}
