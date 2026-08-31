// Package zakat_dto holds the response DTOs of the zakat module. Pure data —
// no functions or methods.
package zakat_dto

import "time"

// GoldPriceResponse is the body of GET /public/zakat/gold-price.
type GoldPriceResponse struct {
	Success  bool      `json:"success"` // false = upstream failed, price is the fallback value
	Price    int       `json:"price"`   // Rp per gram of 1g Antam gold bar
	Source   string    `json:"source"`  // "antam-live" | "fallback"
	CachedAt time.Time `json:"cachedAt"`
}
