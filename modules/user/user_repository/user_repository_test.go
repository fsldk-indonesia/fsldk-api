package user_repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupTestDB membuat database SQLite in-memory dengan skema minimal
// ms_user — hanya kolom yang dipakai SetEmailVerified (lihat
// rapimnas_repository_test.go untuk alasan repository production code
// menghindari fungsi SQL spesifik-dialek seperti NOW() demi tetap portabel
// antara MySQL (produksi) dan SQLite (tes ini)).
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	if err := db.Exec(`CREATE TABLE ms_user (
		userID INTEGER PRIMARY KEY,
		emailVerifiedDate DATETIME,
		updatedDate DATETIME
	)`).Error; err != nil {
		t.Fatalf("create table error = %v", err)
	}
	return db
}

func seedUser(t *testing.T, db *gorm.DB, id int64, verifiedAt *time.Time) {
	t.Helper()
	if err := db.Exec("INSERT INTO ms_user (userID, emailVerifiedDate) VALUES (?, ?)", id, verifiedAt).Error; err != nil {
		t.Fatalf("seedUser() error = %v", err)
	}
}

func fetchEmailVerifiedDate(t *testing.T, db *gorm.DB, id int64) *time.Time {
	t.Helper()
	var got sql.NullTime
	if err := db.Table("ms_user").Select("emailVerifiedDate").Where("userID = ?", id).Take(&got).Error; err != nil {
		t.Fatalf("fetchEmailVerifiedDate() error = %v", err)
	}
	if !got.Valid {
		return nil
	}
	return &got.Time
}

func TestSetEmailVerified_MarksVerifiedWhenPreviouslyUnverified(t *testing.T) {
	db := setupTestDB(t)
	seedUser(t, db, 1, nil)
	repo := NewRepository(db)

	if err := repo.SetEmailVerified(context.Background(), 1, true); err != nil {
		t.Fatalf("SetEmailVerified(true) error = %v", err)
	}

	if got := fetchEmailVerifiedDate(t, db, 1); got == nil {
		t.Error("emailVerifiedDate still NULL after SetEmailVerified(true), want a timestamp")
	}
}

func TestSetEmailVerified_PreservesExistingTimestamp(t *testing.T) {
	db := setupTestDB(t)
	fixed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	seedUser(t, db, 1, &fixed)
	repo := NewRepository(db)

	if err := repo.SetEmailVerified(context.Background(), 1, true); err != nil {
		t.Fatalf("SetEmailVerified(true) error = %v", err)
	}

	got := fetchEmailVerifiedDate(t, db, 1)
	if got == nil || !got.Equal(fixed) {
		t.Errorf("emailVerifiedDate = %v, want unchanged %v (SetEmailVerified must not overwrite an existing verification date)", got, fixed)
	}
}

func TestSetEmailVerified_ClearsDateWhenUnverified(t *testing.T) {
	db := setupTestDB(t)
	now := time.Now()
	seedUser(t, db, 1, &now)
	repo := NewRepository(db)

	if err := repo.SetEmailVerified(context.Background(), 1, false); err != nil {
		t.Fatalf("SetEmailVerified(false) error = %v", err)
	}

	if got := fetchEmailVerifiedDate(t, db, 1); got != nil {
		t.Errorf("emailVerifiedDate = %v, want NULL after SetEmailVerified(false)", got)
	}
}
