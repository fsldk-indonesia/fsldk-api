// Package database menginisialisasi koneksi ke MySQL menggunakan GORM.
package database

import (
	"fmt"
	"net/url"
	"time"

	"fsldk-api/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// New membuka koneksi ke database MySQL tunggal berdasarkan konfigurasi.
// Fungsi ini melakukan Ping untuk memastikan koneksi valid, dan mengembalikan
// error (bukan panic) agar pemanggil dapat menangani kegagalan dengan baik.
func New(cfg config.AppConfig) (*gorm.DB, error) {
	loc := url.QueryEscape(cfg.Timezone)
	if loc == "" {
		loc = "Local"
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=%s&charset=utf8mb4&multiStatements=true",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName, loc,
	)

	logLevel := logger.Warn
	if cfg.AppEnv == "development" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("gagal membuka koneksi database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil koneksi sql: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConn)
	sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConn)
	sqlDB.SetConnMaxLifetime(60 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("gagal ping database %s: %w", cfg.DBName, err)
	}

	return db, nil
}
