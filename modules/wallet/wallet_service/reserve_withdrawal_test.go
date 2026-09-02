package wallet_service

import (
	"context"
	"testing"

	"gorm.io/gorm"

	"fsldk-api/base/apperror"
	"fsldk-api/constants"
	"fsldk-api/modules/wallet/wallet_dto"
	"fsldk-api/modules/wallet/wallet_model"
)

// fakeRepository adalah implementasi wallet_repository.Repository in-memory
// untuk menguji business logic wallet_service tanpa DB sungguhan — tx
// *gorm.DB yang diterima diabaikan (tidak pernah dipakai memanggil GORM).
type fakeRepository struct {
	balance        float64
	insertCalled   bool
	insertedParams wallet_model.LedgerEntryParams
}

func (f *fakeRepository) InsertLedgerEntry(tx *gorm.DB, p wallet_model.LedgerEntryParams) (int64, float64, error) {
	f.insertCalled = true
	f.insertedParams = p
	if p.Direction == constants.LedgerDirectionCredit {
		f.balance += p.Amount
	} else {
		f.balance -= p.Amount
	}
	return 1, f.balance, nil
}

func (f *fakeRepository) LatestBalance(ctx context.Context, campaignID int64) (float64, error) {
	return f.balance, nil
}

func (f *fakeRepository) LatestBalanceForUpdate(tx *gorm.DB, campaignID int64) (float64, error) {
	return f.balance, nil
}

func (f *fakeRepository) SumByEntryType(ctx context.Context, campaignID int64, entryType string) (float64, error) {
	return 0, nil
}

func (f *fakeRepository) SumPendingReserve(ctx context.Context, campaignID int64) (float64, error) {
	return 0, nil
}

func (f *fakeRepository) SumWithdrawnSuccess(ctx context.Context, campaignID int64) (float64, error) {
	return 0, nil
}

func (f *fakeRepository) ListLedger(ctx context.Context, campaignID int64, filter wallet_dto.LedgerListFilter) ([]wallet_model.LedgerEntry, int64, error) {
	return nil, 0, nil
}

func TestReserveWithdrawal_RejectsWhenInsufficientBalance(t *testing.T) {
	repo := &fakeRepository{balance: 50000}
	svc := NewService(repo, nil, nil)

	err := svc.ReserveWithdrawal(nil, 1, 1, 100000, 1, "test")
	if err == nil {
		t.Fatal("expected InsufficientBalance error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeInsufficientBalance {
		t.Fatalf("expected InsufficientBalance app error, got %v", err)
	}
	if repo.insertCalled {
		t.Fatal("InsertLedgerEntry tidak boleh dipanggil saat saldo tidak cukup")
	}
}

func TestReserveWithdrawal_SucceedsWhenBalanceSufficient(t *testing.T) {
	repo := &fakeRepository{balance: 100000}
	svc := NewService(repo, nil, nil)

	err := svc.ReserveWithdrawal(nil, 1, 1, 60000, 1, "test")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if !repo.insertCalled {
		t.Fatal("expected InsertLedgerEntry to be called")
	}
	if repo.insertedParams.EntryType != constants.LedgerEntryWithdrawalReserve {
		t.Fatalf("expected entryType WITHDRAWAL_RESERVE, got %s", repo.insertedParams.EntryType)
	}
	if repo.insertedParams.Direction != constants.LedgerDirectionDebit {
		t.Fatalf("expected direction DEBIT, got %s", repo.insertedParams.Direction)
	}
}

func TestReserveWithdrawal_AllowsExactBalance(t *testing.T) {
	repo := &fakeRepository{balance: 50000}
	svc := NewService(repo, nil, nil)

	if err := svc.ReserveWithdrawal(nil, 1, 1, 50000, 1, "test"); err != nil {
		t.Fatalf("expected success when amount == available balance, got error: %v", err)
	}
}
