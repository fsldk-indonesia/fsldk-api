// Package migrations menjalankan migration SQL secara berurutan dan mencatat
// migration yang telah diterapkan pada tabel schema_migrations.
package migrations

import (
	"embed"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"
)

//go:embed *.up.sql
var files embed.FS

// Run menerapkan seluruh migration *.up.sql yang belum dijalankan, termasuk
// seed data (role, permission, kategori, konten awal, akun Super Admin —
// lihat 0002_seed.up.sql dan 0003_seed_admin.up.sql).
func Run(db *gorm.DB) error {
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) PRIMARY KEY,
		appliedAt DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`).Error; err != nil {
		return fmt.Errorf("gagal membuat tabel schema_migrations: %w", err)
	}

	entries, err := files.ReadDir(".")
	if err != nil {
		return err
	}

	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".up.sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		var count int64
		if err := db.Raw("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", name).Scan(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}

		content, err := files.ReadFile(name)
		if err != nil {
			return err
		}
		if err := db.Exec(string(content)).Error; err != nil {
			return fmt.Errorf("gagal menjalankan migration %s: %w", name, err)
		}
		if err := db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", name).Error; err != nil {
			return err
		}
		fmt.Printf("[migration] diterapkan: %s\n", name)
	}
	return nil
}
