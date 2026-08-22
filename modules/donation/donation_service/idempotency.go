package donation_service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"time"
)

// idempotencyFallbackWindow adalah lebar jendela waktu dipakai fallback
// idempotency key ketika klien tidak mengirim key sendiri — permintaan
// dengan donorEmail+campaignID+amount yang sama dalam window ini
// menghasilkan key identik, mencegah donasi ganda akibat double-click/
// refresh tanpa mengharuskan client mengirim UUID sendiri.
const idempotencyFallbackWindow = 5 * time.Minute

// fallbackIdempotencyKey menghasilkan idempotency key deterministik dari
// identitas donasi + jendela waktu saat ini.
func fallbackIdempotencyKey(donorEmail string, campaignID int64, amount float64, now time.Time) string {
	window := now.Unix() / int64(idempotencyFallbackWindow.Seconds())
	raw := fmt.Sprintf("%s:%d:%d:%d", strings.ToLower(strings.TrimSpace(donorEmail)), campaignID, roundRupiah(amount), window)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// roundRupiah membulatkan nominal ke Rupiah utuh — IDR tidak punya sub-unit
// praktis, dan formula MDR (pkg/bisatopup) beroperasi pada integer Rupiah.
func roundRupiah(amount float64) int64 {
	return int64(math.Round(amount))
}
