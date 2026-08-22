// Package wallet_dto memuat request/response modul wallet. Murni struct
// data.
package wallet_dto

import "time"

// BalanceResponse adalah ringkasan saldo satu campaign.
type BalanceResponse struct {
	AvailableBalance float64 `json:"availableBalance"`
	PendingBalance   float64 `json:"pendingBalance"`
	TotalCollected   float64 `json:"totalCollected"`
	TotalWithdrawn   float64 `json:"totalWithdrawn"`
}

// LedgerListItem adalah satu baris riwayat ledger untuk ditampilkan ke
// pemilik campaign/admin.
type LedgerListItem struct {
	LedgerID      int64     `json:"ledgerID"`
	EntryType     string    `json:"entryType"`
	Direction     string    `json:"direction"`
	Amount        float64   `json:"amount"`
	BalanceAfter  float64   `json:"balanceAfter"`
	ReferenceType string    `json:"referenceType"`
	ReferenceID   int64     `json:"referenceID"`
	Note          string    `json:"note,omitempty"`
	CreatedDate   time.Time `json:"createdDate"`
}

// LedgerListFilter menampung filter untuk ListLedger.
type LedgerListFilter struct {
	EntryType string
	DateFrom  *time.Time
	DateTo    *time.Time
	Page      int
	Limit     int
}
