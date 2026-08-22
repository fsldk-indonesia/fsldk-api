package wallet_repository

import (
	"testing"

	"fsldk-api/constants"
)

func TestApplyDirection_Credit(t *testing.T) {
	got := applyDirection(100000, 50000, constants.LedgerDirectionCredit)
	if got != 150000 {
		t.Fatalf("applyDirection CREDIT = %v, want 150000", got)
	}
}

func TestApplyDirection_Debit(t *testing.T) {
	got := applyDirection(100000, 30000, constants.LedgerDirectionDebit)
	if got != 70000 {
		t.Fatalf("applyDirection DEBIT = %v, want 70000", got)
	}
}

// TestApplyDirection_Sequence mensimulasikan satu campaign: donasi masuk,
// withdrawal direservasi, lalu withdrawal itu gagal dan reservasinya
// dilepas — saldo akhir harus kembali seperti sebelum reservasi.
func TestApplyDirection_Sequence(t *testing.T) {
	balance := 0.0
	balance = applyDirection(balance, 100000, constants.LedgerDirectionCredit) // DONATION_CREDIT
	balance = applyDirection(balance, 40000, constants.LedgerDirectionDebit)   // WITHDRAWAL_RESERVE
	balance = applyDirection(balance, 40000, constants.LedgerDirectionCredit)  // WITHDRAWAL_RELEASE (gagal)

	if balance != 100000 {
		t.Fatalf("balance setelah credit/debit/credit = %v, want 100000", balance)
	}
}
