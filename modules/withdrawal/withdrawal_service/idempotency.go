package withdrawal_service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"time"
)

// idempotencyFallbackWindow adalah lebar jendela waktu dipakai fallback
// idempotency key ketika klien tidak mengirim key sendiri — sama alasannya
// dengan donation_service (mencegah request ganda akibat double-click/refresh).
const idempotencyFallbackWindow = 5 * time.Minute

// fallbackIdempotencyKey menghasilkan idempotency key deterministik dari
// identitas withdrawal + jendela waktu saat ini.
func fallbackIdempotencyKey(campaignID int64, amount float64, now time.Time) string {
	window := now.Unix() / int64(idempotencyFallbackWindow.Seconds())
	raw := fmt.Sprintf("%d:%d:%d", campaignID, roundRupiah(amount), window)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// roundRupiah membulatkan nominal ke Rupiah utuh.
func roundRupiah(amount float64) int64 {
	return int64(math.Round(amount))
}

// truncateNoEllipsis memotong s ke maksimal max karakter tanpa suffix "..."
// — dipakai untuk field remark Bisabiller yang membatasi panjang string ketat.
func truncateNoEllipsis(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}
