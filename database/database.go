// Package database menginisialisasi koneksi ke MySQL menggunakan sqlx.
package database

import (
	"fmt"
	"net/url"
	"time"

	"fsldk-api/config"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// New membuka koneksi ke database MySQL tunggal berdasarkan konfigurasi.
// Fungsi ini melakukan Ping untuk memastikan koneksi valid, dan mengembalikan
// error (bukan panic) agar pemanggil dapat menangani kegagalan dengan baik.
func New(cfg config.AppConfig) (*sqlx.DB, error) {
	loc := url.QueryEscape(cfg.Timezone)
	if loc == "" {
		loc = "Local"
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=%s&charset=utf8mb4&multiStatements=true",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName, loc,
	)

	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("gagal membuka koneksi database: %w", err)
	}

	db.SetMaxOpenConns(cfg.DBMaxOpenConn)
	db.SetMaxIdleConns(cfg.DBMaxIdleConn)
	db.SetConnMaxLifetime(60 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("gagal ping database %s: %w", cfg.DBName, err)
	}

	return db, nil
}
