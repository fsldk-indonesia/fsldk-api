package donation_service

import (
	"testing"
	"time"
)

func TestFallbackIdempotencyKey_SameInputsSameWindow_SameKey(t *testing.T) {
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	k1 := fallbackIdempotencyKey("donor@example.com", 1, 50000, now)
	k2 := fallbackIdempotencyKey("donor@example.com", 1, 50000, now.Add(1*time.Minute))
	if k1 != k2 {
		t.Fatalf("expected same key within the fallback window, got %q vs %q", k1, k2)
	}
}

func TestFallbackIdempotencyKey_DifferentWindow_DifferentKey(t *testing.T) {
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	k1 := fallbackIdempotencyKey("donor@example.com", 1, 50000, now)
	k2 := fallbackIdempotencyKey("donor@example.com", 1, 50000, now.Add(10*time.Minute))
	if k1 == k2 {
		t.Fatal("expected different key once the fallback window elapses")
	}
}

func TestFallbackIdempotencyKey_DifferentAmount_DifferentKey(t *testing.T) {
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	k1 := fallbackIdempotencyKey("donor@example.com", 1, 50000, now)
	k2 := fallbackIdempotencyKey("donor@example.com", 1, 75000, now)
	if k1 == k2 {
		t.Fatal("expected different key for a different amount")
	}
}

func TestFallbackIdempotencyKey_DifferentCampaign_DifferentKey(t *testing.T) {
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	k1 := fallbackIdempotencyKey("donor@example.com", 1, 50000, now)
	k2 := fallbackIdempotencyKey("donor@example.com", 2, 50000, now)
	if k1 == k2 {
		t.Fatal("expected different key for a different campaign")
	}
}

func TestFallbackIdempotencyKey_EmailCaseInsensitive(t *testing.T) {
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	k1 := fallbackIdempotencyKey("Donor@Example.com", 1, 50000, now)
	k2 := fallbackIdempotencyKey("donor@example.com", 1, 50000, now)
	if k1 != k2 {
		t.Fatal("expected key to be case-insensitive on donor email")
	}
}

func TestRoundRupiah(t *testing.T) {
	cases := map[float64]int64{
		50000:   50000,
		50000.4: 50000,
		50000.5: 50001,
		999.99:  1000,
	}
	for in, want := range cases {
		if got := roundRupiah(in); got != want {
			t.Errorf("roundRupiah(%v) = %d, want %d", in, got, want)
		}
	}
}
