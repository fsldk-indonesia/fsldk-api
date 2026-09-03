// Package wallet_model memuat entitas modul wallet. Murni struct data.
package wallet_model

import (
	"database/sql"
	"time"
)

// LedgerEntry merepresentasikan satu baris tr_wallet_ledger.
type LedgerEntry struct {
	LedgerID        int64         `gorm:"column:ledgerID;primaryKey"`
	CampaignID      int64         `gorm:"column:campaignID"`
	EntryType       string        `gorm:"column:entryType"`
	Direction       string        `gorm:"column:direction"`
	Amount          float64       `gorm:"column:amount"`
	BalanceAfter    float64       `gorm:"column:balanceAfter"`
	ReferenceType   string        `gorm:"column:referenceType"`
	ReferenceID     int64         `gorm:"column:referenceID"`
	Note            sql.NullString `gorm:"column:note"`
	CreatedByUserID sql.NullInt64  `gorm:"column:createdByUserID"`
	CreatedDate     time.Time      `gorm:"column:createdDate"`
}

// LedgerEntryParams menampung data untuk membuat baris tr_wallet_ledger
// baru lewat InsertLedgerEntry — satu-satunya jalur tulis ke ledger,
// lihat wallet_service.
type LedgerEntryParams struct {
	CampaignID      int64
	EntryType       string
	Direction       string
	Amount          float64
	ReferenceType   string
	ReferenceID     int64
	Note            sql.NullString
	CreatedByUserID sql.NullInt64
}
