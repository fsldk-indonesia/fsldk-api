// Package dbretry menyediakan retry helper untuk transaksi yang terkena
// deadlock/lock-wait-timeout MySQL (error 1213/1205) — dipakai titik-titik
// yang mengunci baris ms_campaign yang sama dari banyak transaksi pendek
// bersamaan (donation callback, withdrawal callback). Deadlock InnoDB pada
// skenario ini normal di bawah kontensi tinggi, bukan indikasi bug: transaksi
// yang jadi "deadlock victim" di-rollback utuh oleh InnoDB, aman diulang.
package dbretry

import (
	"errors"
	"math/rand"
	"time"

	"github.com/go-sql-driver/mysql"
)

const maxAttempts = 8

// Do menjalankan fn, mengulang otomatis (exponential backoff 20ms→300ms +
// jitter) hanya untuk error 1213 (deadlock) / 1205 (lock wait timeout) —
// error lain (termasuk error bisnis seperti apperror.AppError) langsung
// dikembalikan tanpa retry.
func Do(fn func() error) error {
	var err error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		err = fn()
		if err == nil || !isRetryable(err) {
			return err
		}
		backoff := 20 * time.Millisecond * time.Duration(1<<attempt)
		if backoff > 300*time.Millisecond {
			backoff = 300 * time.Millisecond
		}
		time.Sleep(backoff + time.Duration(rand.Intn(20))*time.Millisecond)
	}
	return err
}

func isRetryable(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1213 || mysqlErr.Number == 1205
	}
	return false
}
