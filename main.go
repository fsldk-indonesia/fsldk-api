// Command fsldk-api adalah REST API untuk Website FSLDK Indonesia.
package main

import (
	"log"
	"os"

	"fsldk-api/config"
	"fsldk-api/database"
	"fsldk-api/migrations"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("gagal memuat konfigurasi: %v", err)
	}

	if cfg.Timezone != "" {
		_ = os.Setenv("TZ", cfg.Timezone)
	}

	db, err := database.New(cfg)
	if err != nil {
		log.Fatalf("gagal terhubung ke database: %v", err)
	}
	defer func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}()

	// Jalankan migration & seed data awal (termasuk akun Super Admin — lihat
	// migrations/0003_seed_admin.up.sql).
	if err := migrations.Run(db); err != nil {
		log.Fatalf("gagal menjalankan migration: %v", err)
	}

	engine := setupRouter(db, cfg)

	addr := cfg.AppHost + ":" + cfg.AppPort
	log.Printf("FSLDK API berjalan pada http://%s (env: %s)", addr, cfg.AppEnv)
	if err := engine.Run(addr); err != nil {
		log.Fatalf("server berhenti: %v", err)
	}
}
