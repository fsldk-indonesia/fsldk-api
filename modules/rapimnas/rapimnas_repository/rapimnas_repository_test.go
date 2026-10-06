package rapimnas_repository

import (
	"context"
	"testing"

	"fsldk-api/modules/rapimnas/rapimnas_model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupTestDB membuat database SQLite in-memory dengan skema yang setara
// dengan migrations/0046_rapimnas.up.sql (disederhanakan ke sintaks SQLite —
// lihat "Design decisions" item 4/5 di kepala plan ini untuk alasan
// repository production code TIDAK memakai SQL mentah apa pun selain
// DELETE FROM, supaya tetap portabel antara MySQL (produksi) dan SQLite (tes
// ini)). PRAGMA foreign_keys=ON supaya pelanggaran FK rundown_event->rundown_day
// benar-benar gagal di Task 7.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	// SQLite in-memory: satu koneksi saja, supaya seluruh query dalam test
	// (dan di dalam transaksi repository) benar-benar melihat database yang
	// sama, bukan instance in-memory terpisah per koneksi.
	sqlDB.SetMaxOpenConns(1)

	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatalf("enable foreign_keys error = %v", err)
	}

	ddl := []string{
		`CREATE TABLE ms_rapimnas_setting (
			id INTEGER PRIMARY KEY,
			heroBadgeText TEXT, heroTitle TEXT, heroDateRangeText TEXT, heroTaglineQuote TEXT, heroImageUrl TEXT,
			countdownTargetDate DATETIME,
			feature1IconKey TEXT, feature1Title TEXT, feature1Desc TEXT,
			feature2IconKey TEXT, feature2Title TEXT, feature2Desc TEXT,
			ctaTitle TEXT, ctaDescription TEXT, ctaButtonLabel TEXT, ctaMascotImageUrl TEXT,
			footerContactEmail TEXT, footerCopyrightText TEXT, footerIgHandle TEXT, footerIgUrl TEXT,
			footerTiktokHandle TEXT, footerTiktokUrl TEXT,
			jadwalHeaderSubtitle TEXT,
			tentangTaglineQuote TEXT, tentangDescParagraph1 TEXT, tentangDescParagraph2 TEXT, tentangVisiText TEXT,
			tentangMisiJSON TEXT, tentangTujuanJSON TEXT, tentangKegiatanJSON TEXT,
			pesertaEarlyBirdDateRange TEXT, pesertaRegulerDateRange TEXT,
			pesertaHargaNonSemarangEarlyBird INTEGER, pesertaHargaNonSemarangReguler INTEGER,
			pesertaHargaSemarangEarlyBird INTEGER, pesertaHargaSemarangReguler INTEGER,
			pesertaBankName TEXT, pesertaBankAccountNumber TEXT, pesertaBankAccountHolder TEXT,
			pesertaGuidebookUrl TEXT, pesertaGoogleFormUrl TEXT, pesertaMapEmbedUrl TEXT,
			panitiaIsOpen BOOLEAN, panitiaClosedMessage TEXT,
			updatedDate DATETIME, updatedBy INTEGER
		)`,
		`CREATE TABLE ms_rapimnas_gallery_image (id INTEGER PRIMARY KEY AUTOINCREMENT, imageUrl TEXT NOT NULL, sortOrder INTEGER NOT NULL DEFAULT 0)`,
		`CREATE TABLE ms_rapimnas_home_card (id INTEGER PRIMARY KEY AUTOINCREMENT, iconKey TEXT, title TEXT, description TEXT, sortOrder INTEGER NOT NULL DEFAULT 0)`,
		`CREATE TABLE ms_rapimnas_rundown_day (id INTEGER PRIMARY KEY AUTOINCREMENT, dayLabel TEXT, dateText TEXT, sortOrder INTEGER NOT NULL DEFAULT 0)`,
		`CREATE TABLE ms_rapimnas_rundown_event (id INTEGER PRIMARY KEY AUTOINCREMENT, dayID INTEGER NOT NULL, time TEXT, title TEXT, description TEXT, venue TEXT, sortOrder INTEGER NOT NULL DEFAULT 0, FOREIGN KEY(dayID) REFERENCES ms_rapimnas_rundown_day(id))`,
		`CREATE TABLE ms_rapimnas_resource (id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT, description TEXT, iconKey TEXT, url TEXT, buttonLabel TEXT, isVisible BOOLEAN NOT NULL DEFAULT 1, sortOrder INTEGER NOT NULL DEFAULT 0)`,
		`CREATE TABLE ms_rapimnas_pickup_location (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, type TEXT, description TEXT, mapLink TEXT, sortOrder INTEGER NOT NULL DEFAULT 0)`,
		`CREATE TABLE ms_rapimnas_contact (id INTEGER PRIMARY KEY AUTOINCREMENT, contactType TEXT NOT NULL, name TEXT, phoneNumber TEXT, sortOrder INTEGER NOT NULL DEFAULT 0)`,
	}
	for _, stmt := range ddl {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("create table error = %v\nSQL: %s", err, stmt)
		}
	}
	if err := db.Exec("INSERT INTO ms_rapimnas_setting (id) VALUES (1)").Error; err != nil {
		t.Fatalf("seed singleton row error = %v", err)
	}
	return db
}

func TestSaveThenGet_ReplacesChildCollectionsInSubmittedOrder(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	agg := &rapimnas_model.Aggregate{
		Setting: rapimnas_model.Setting{HeroTitle: "Rapimnas 2026"},
		GalleryImages: []rapimnas_model.GalleryImage{
			{ImageUrl: "a.jpg"}, {ImageUrl: "b.jpg"}, {ImageUrl: "c.jpg"},
		},
		RundownDays: []rapimnas_model.RundownDay{
			{ID: 0, DayLabel: "Hari 1"},
			{ID: 1, DayLabel: "Hari 2"},
		},
		RundownEvents: []rapimnas_model.RundownEvent{
			{DayID: 1, Title: "Event B"},
			{DayID: 0, Title: "Event A"},
		},
	}
	if err := repo.Save(ctx, agg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := repo.Get(ctx)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Setting.HeroTitle != "Rapimnas 2026" {
		t.Errorf("Setting.HeroTitle = %q, want %q", got.Setting.HeroTitle, "Rapimnas 2026")
	}
	if len(got.GalleryImages) != 3 || got.GalleryImages[0].ImageUrl != "a.jpg" || got.GalleryImages[2].ImageUrl != "c.jpg" {
		t.Fatalf("GalleryImages = %+v, want 3 images in submitted order [a.jpg b.jpg c.jpg]", got.GalleryImages)
	}

	var dayAID, dayBID int64
	for _, d := range got.RundownDays {
		switch d.DayLabel {
		case "Hari 1":
			dayAID = d.ID
		case "Hari 2":
			dayBID = d.ID
		}
	}
	if dayAID == 0 || dayBID == 0 {
		t.Fatalf("RundownDays missing expected rows: %+v", got.RundownDays)
	}
	for _, e := range got.RundownEvents {
		if e.Title == "Event A" && e.DayID != dayAID {
			t.Errorf("Event A.DayID = %d, want %d (Hari 1's real id)", e.DayID, dayAID)
		}
		if e.Title == "Event B" && e.DayID != dayBID {
			t.Errorf("Event B.DayID = %d, want %d (Hari 2's real id)", e.DayID, dayBID)
		}
	}

	// Simpan ulang dengan urutan & jumlah berbeda — membuktikan Save()
	// mengganti seluruh koleksi (delete+recreate), bukan menambah (append-only).
	agg2 := &rapimnas_model.Aggregate{
		Setting: rapimnas_model.Setting{HeroTitle: "Rapimnas 2026 (updated)"},
		GalleryImages: []rapimnas_model.GalleryImage{
			{ImageUrl: "c.jpg"}, {ImageUrl: "a.jpg"},
		},
	}
	if err := repo.Save(ctx, agg2); err != nil {
		t.Fatalf("second Save() error = %v", err)
	}
	got2, err := repo.Get(ctx)
	if err != nil {
		t.Fatalf("second Get() error = %v", err)
	}
	if len(got2.GalleryImages) != 2 || got2.GalleryImages[0].ImageUrl != "c.jpg" || got2.GalleryImages[1].ImageUrl != "a.jpg" {
		t.Fatalf("GalleryImages after second Save() = %+v, want exactly [c.jpg a.jpg] (replace, not append)", got2.GalleryImages)
	}
	if len(got2.RundownDays) != 0 || len(got2.RundownEvents) != 0 {
		t.Errorf("RundownDays/RundownEvents after second Save() (which submitted none) = %+v / %+v, want empty", got2.RundownDays, got2.RundownEvents)
	}
}
