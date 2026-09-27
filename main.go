// Command fsldk-api adalah REST API untuk Website FSLDK Indonesia.
package main

import (
	"log"
	"os"

	"fsldk-api/config"
	"fsldk-api/database"
	"fsldk-api/migrations"
	"fsldk-api/pkg/upload"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "backfill-thumbnails" {
		runBackfillThumbnails()
		return
	}

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

// runBackfillThumbnails adalah operasi satu-kali (`go run . backfill-thumbnails`
// atau binary hasil build dijalankan dengan argumen yang sama) untuk gambar
// yang sudah terupload sebelum fitur resize-on-save ini ada — men-generate
// varian "_thumb" yang belum punya, dan meng-cap file utamanya di tempat
// kalau lebih besar dari batas display (lihat upload.Uploader.BackfillThumbnails).
// Berjalan murni di atas folder disk, tidak butuh koneksi database.
func runBackfillThumbnails() {
	u := upload.NewUploader("assets/uploads", "")

	result, err := u.BackfillThumbnails()
	if err != nil {
		log.Fatalf("backfill-thumbnails gagal membaca folder uploads: %v", err)
	}

	log.Printf("backfill-thumbnails selesai: %d diproses, %d dilewati (sudah punya thumb)", len(result.Processed), len(result.Skipped))
	for name, ferr := range result.Failed {
		log.Printf("  gagal memproses %s: %v", name, ferr)
	}
	if len(result.Failed) > 0 {
		log.Fatalf("backfill-thumbnails selesai dengan %d kegagalan", len(result.Failed))
	}
}
