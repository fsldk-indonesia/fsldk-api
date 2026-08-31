// Package financeformat_repository is the data access layer for the financeformat module (GORM).
package financeformat_repository

import (
	"context"
	"errors"

	"fsldk-api/modules/financeformat/financeformat_dto"
	"fsldk-api/modules/financeformat/financeformat_model"
)

// ErrNotFound is returned when a finance format cannot be found.
var ErrNotFound = errors.New("finance format not found")

// Repository is the data access contract for finance formats.
type Repository interface {
	List(ctx context.Context, f financeformat_dto.Filter) ([]financeformat_model.FinanceFormat, int64, error)
	FindByID(ctx context.Context, id int64) (financeformat_model.FinanceFormat, error)
	Create(ctx context.Context, m financeformat_model.FinanceFormat, actorID int64) (int64, error)
	Update(ctx context.Context, id int64, m financeformat_model.FinanceFormat, actorID int64) error
	SetActive(ctx context.Context, id int64, isActive bool, actorID int64) error
	Delete(ctx context.Context, id int64) error

	ListFormatTypes(ctx context.Context) ([]financeformat_model.FormatType, error)
	FormatTypeExists(ctx context.Context, id int64) (bool, error)
}
