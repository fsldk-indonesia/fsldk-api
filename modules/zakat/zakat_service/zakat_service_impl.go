package zakat_service

import (
	"context"

	"fsldk-api/modules/zakat/zakat_dto"
	"fsldk-api/pkg/goldprice"
)

// ServiceImpl is the Service implementation. It calls *goldprice.Client, not a
// repository — this module never touches the DB.
type ServiceImpl struct{ gold *goldprice.Client }

// NewService builds the zakat Service.
func NewService(gold *goldprice.Client) Service { return &ServiceImpl{gold: gold} }

func (s *ServiceImpl) GoldPrice(ctx context.Context, forceRefresh bool) zakat_dto.GoldPriceResponse {
	p := s.gold.Get(ctx, forceRefresh)
	return zakat_dto.GoldPriceResponse{
		Success:  p.Success,
		Price:    p.Price,
		Source:   p.Source,
		CachedAt: p.CachedAt,
	}
}
