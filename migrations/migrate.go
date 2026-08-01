// Package migrations menjalankan migration SQL secara berurutan dan mencatat
// migration yang telah diterapkan pada tabel schema_migrations.
package migrations

import (
	"embed"
	"fmt"
	"os"
	"sort"
	"strings"

	"fsldk-api/base/security"
	"fsldk-api/config"
	"fsldk-api/constants"

	"gorm.io/gorm"
)

//go:embed *.up.sql
var files embed.FS

// Run menerapkan seluruh migration *.up.sql yang belum dijalankan.
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

// EnsureSuperAdmin membuat akun Super Admin awal bila belum ada.
// Kredensial diambil dari environment (SEED_ADMIN_EMAIL / SEED_ADMIN_PASSWORD),
// dengan nilai default yang aman untuk pengembangan.
func EnsureSuperAdmin(db *gorm.DB, cfg config.AppConfig) error {
	email := getenvDefault("SEED_ADMIN_EMAIL", "admin@fsldk-indonesia.com")
	password := getenvDefault("SEED_ADMIN_PASSWORD", "Admin@123")
	name := getenvDefault("SEED_ADMIN_NAME", "Admin FSLDK")

	var count int64
	if err := db.Raw("SELECT COUNT(*) FROM ms_user WHERE email = ?", email).Scan(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	var roleID int64
	if err := db.Raw("SELECT roleID FROM ms_role WHERE roleName = ?", constants.RoleSuperAdmin).Scan(&roleID).Error; err != nil {
		return err
	}
	if roleID == 0 {
		return fmt.Errorf("role Super Admin belum ter-seed")
	}

	hashed, err := security.HashPassword(password)
	if err != nil {
		return err
	}

	err = db.Exec(`INSERT INTO ms_user
		(roleID, fullName, email, password, emailVerifiedDate, mustChangePassword, isActive, createdDate)
		VALUES (?, ?, ?, ?, NOW(), 0, 1, NOW())`,
		roleID, name, email, hashed).Error
	if err != nil {
		return err
	}
	fmt.Printf("[seed] Super Admin dibuat: %s (password default: %s — segera ganti)\n", email, password)
	return nil
}

func getenvDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
