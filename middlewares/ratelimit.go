package middlewares

import (
	"sync"
	"time"

	"fsldk-api/base/apperror"
	"fsldk-api/base/httphelper"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// visitor menyimpan limiter beserta waktu akses terakhir untuk keperluan cleanup.
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// keyedLimiter adalah penyimpanan limiter per-kunci (mis. per-IP) dengan
// pembersihan otomatis entri lama.
type keyedLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	limit    rate.Limit
	burst    int
}

func newKeyedLimiter(r rate.Limit, burst int) *keyedLimiter {
	kl := &keyedLimiter{visitors: make(map[string]*visitor), limit: r, burst: burst}
	go kl.cleanupLoop()
	return kl
}

func (kl *keyedLimiter) get(key string) *rate.Limiter {
	kl.mu.Lock()
	defer kl.mu.Unlock()
	v, ok := kl.visitors[key]
	if !ok {
		lim := rate.NewLimiter(kl.limit, kl.burst)
		kl.visitors[key] = &visitor{limiter: lim, lastSeen: time.Now()}
		return lim
	}
	v.lastSeen = time.Now()
	return v.limiter
}

func (kl *keyedLimiter) cleanupLoop() {
	for {
		time.Sleep(5 * time.Minute)
		kl.mu.Lock()
		for k, v := range kl.visitors {
			if time.Since(v.lastSeen) > 10*time.Minute {
				delete(kl.visitors, k)
			}
		}
		kl.mu.Unlock()
	}
}

// RateLimit membatasi jumlah permintaan per IP. Parameter perMinute adalah
// jumlah permintaan yang diizinkan per menit, burst adalah lonjakan maksimum.
func RateLimit(perMinute float64, burst int) gin.HandlerFunc {
	kl := newKeyedLimiter(rate.Limit(perMinute/60.0), burst)
	return func(c *gin.Context) {
		if !kl.get(c.ClientIP()).Allow() {
			httphelper.Error(c, apperror.TooManyRequests(""))
			return
		}
		c.Next()
	}
}
