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

	"github.com/jmoiron/sqlx"
)

//go:embed *.up.sql
var files embed.FS

// Run menerapkan seluruh migration *.up.sql yang belum dijalankan.
func Run(db *sqlx.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) PRIMARY KEY,
		appliedAt DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`); err != nil {
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
		var count int
		if err := db.Get(&count, "SELECT COUNT(*) FROM schema_migrations WHERE version = ?", name); err != nil {
			return err
		}
		if count > 0 {
			continue
		}

		content, err := files.ReadFile(name)
		if err != nil {
			return err
		}
		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("gagal menjalankan migration %s: %w", name, err)
		}
		if _, err := db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", name); err != nil {
			return err
		}
		fmt.Printf("[migration] diterapkan: %s\n", name)
	}
	return nil
}

// EnsureSuperAdmin membuat akun Super Admin awal bila belum ada.
// Kredensial diambil dari environment (SEED_ADMIN_EMAIL / SEED_ADMIN_PASSWORD),
// dengan nilai default yang aman untuk pengembangan.
func EnsureSuperAdmin(db *sqlx.DB, cfg config.AppConfig) error {
	email := getenvDefault("SEED_ADMIN_EMAIL", "admin@fsldk-indonesia.com")
	password := getenvDefault("SEED_ADMIN_PASSWORD", "Admin@123")
	name := getenvDefault("SEED_ADMIN_NAME", "Admin FSLDK")

	var count int
	if err := db.Get(&count, "SELECT COUNT(*) FROM ms_user WHERE email = ?", email); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	var roleID int64
	if err := db.Get(&roleID, "SELECT roleID FROM ms_role WHERE roleName = ?", constants.RoleSuperAdmin); err != nil {
		return fmt.Errorf("role Super Admin belum ter-seed: %w", err)
	}

	hashed, err := security.HashPassword(password)
	if err != nil {
		return err
	}

	_, err = db.Exec(`INSERT INTO ms_user
		(roleID, fullName, email, password, emailVerifiedDate, mustChangePassword, isActive, createdDate)
		VALUES (?, ?, ?, ?, NOW(), 0, 1, NOW())`,
		roleID, name, email, hashed)
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
