// Package financeformat_service holds financeformat module business logic.
package financeformat_service

import (
	"context"

	"fsldk-api/base/dto"
	"fsldk-api/modules/financeformat/financeformat_dto"
	"fsldk-api/modules/financeformat/financeformat_model"
)

// Service is the business logic contract for finance formats.
type Service interface {
	PublicList(ctx context.Context) (financeformat_dto.PublicListResponse, error)
	PrepareDownload(ctx context.Context, id int64) (localPath string, downloadName string, err error)
	CMSList(ctx context.Context, q dto.ListQuery, f financeformat_dto.Filter) ([]financeformat_model.FinanceFormat, int, error)
	Get(ctx context.Context, id int64) (financeformat_model.FinanceFormat, error)
	FormatTypes(ctx context.Context) ([]financeformat_model.FormatType, error)

	Create(ctx context.Context, req financeformat_dto.Request, actorID int64) (financeformat_model.FinanceFormat, error)
	Update(ctx context.Context, id int64, req financeformat_dto.Request, actorID int64) (financeformat_model.FinanceFormat, error)
	SetActive(ctx context.Context, id int64, isActive bool, actorID int64) error
	Delete(ctx context.Context, id int64) error
}
