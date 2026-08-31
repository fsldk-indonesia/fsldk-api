// Package financeformat_model holds financeformat module entities. Pure data, no methods.
package financeformat_model

import "time"

// FinanceFormat represents one ms_finance_format row, joined to
// lk_finance_format_type for the category name (read-only "->" column).
type FinanceFormat struct {
	FinanceFormatID int64     `gorm:"column:financeFormatID;primaryKey" json:"financeFormatID"`
	FileName        string    `gorm:"column:fileName" json:"fileName"`
	FileURL         string    `gorm:"column:fileURL" json:"fileURL"`
	FormatTypeID    int64     `gorm:"column:formatTypeID" json:"formatTypeID"`
	FormatTypeName  string    `gorm:"column:formatTypeName;->" json:"formatTypeName"`
	IsActive        bool      `gorm:"column:isActive" json:"isActive"`
	CreatedDate     time.Time `gorm:"column:createdDate" json:"createdDate"`
}

// FormatType represents one lk_finance_format_type row (9 fixed categories).
type FormatType struct {
	FormatTypeID   int64  `gorm:"column:formatTypeID;primaryKey" json:"formatTypeID"`
	FormatTypeName string `gorm:"column:formatTypeName" json:"formatTypeName"`
	SortOrder      int    `gorm:"column:sortOrder" json:"sortOrder"`
}
