// Package zakat_service holds the business logic of the zakat module. The
// module is DB-less (no model/repository, like modules/upload): its only job
// is proxying the gold price.
package zakat_service

import (
	"context"

	"fsldk-api/modules/zakat/zakat_dto"
)

// Service is the zakat business-logic contract.
type Service interface {
	// GoldPrice returns the current 1g Antam gold-bar price. It always
	// succeeds with a valid body: on upstream failure Success is false and
	// Price carries the configured fallback.
	GoldPrice(ctx context.Context, forceRefresh bool) zakat_dto.GoldPriceResponse
}
