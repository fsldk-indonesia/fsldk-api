// Package rapimnas_model memuat entitas modul rapimnas (microsite RAPIMNAS 1
// FSLDK Indonesia 2026 — satu baris singleton pengaturan + 6 tabel anak
// ber-sortOrder). Seluruhnya murni struct data (tanpa function/method),
// kecuali konstanta SingletonID dan ContactType*.
package rapimnas_model

import "time"

// SingletonID adalah satu-satunya id baris yang pernah ada di
// ms_rapimnas_setting — sama seperti welcomepopup_model.SingletonID.
const SingletonID int64 = 1

// Kode contactType pada ms_rapimnas_contact.
const (
	ContactTypePesertaCP = "peserta_cp"
	ContactTypeFooterWA  = "footer_wa"
)

// Setting merepresentasikan satu baris ms_rapimnas_setting.
type Setting struct {
	ID                               int64      `gorm:"column:id;primaryKey"`
	HeroBadgeText                    string     `gorm:"column:heroBadgeText"`
	HeroTitle                        string     `gorm:"column:heroTitle"`
	HeroDateRangeText                string     `gorm:"column:heroDateRangeText"`
	HeroTaglineQuote                 string     `gorm:"column:heroTaglineQuote"`
	HeroImageUrl                     *string    `gorm:"column:heroImageUrl"`
	CountdownTargetDate              *time.Time `gorm:"column:countdownTargetDate"`
	Feature1IconKey                  string     `gorm:"column:feature1IconKey"`
	Feature1Title                    string     `gorm:"column:feature1Title"`
	Feature1Desc                     string     `gorm:"column:feature1Desc"`
	Feature2IconKey                  string     `gorm:"column:feature2IconKey"`
	Feature2Title                    string     `gorm:"column:feature2Title"`
	Feature2Desc                     string     `gorm:"column:feature2Desc"`
	CtaTitle                         string     `gorm:"column:ctaTitle"`
	CtaDescription                   string     `gorm:"column:ctaDescription"`
	CtaButtonLabel                   string     `gorm:"column:ctaButtonLabel"`
	CtaMascotImageUrl                *string    `gorm:"column:ctaMascotImageUrl"`
	FooterContactEmail               string     `gorm:"column:footerContactEmail"`
	FooterCopyrightText              string     `gorm:"column:footerCopyrightText"`
	FooterIgHandle                   string     `gorm:"column:footerIgHandle"`
	FooterIgUrl                      string     `gorm:"column:footerIgUrl"`
	FooterTiktokHandle               string     `gorm:"column:footerTiktokHandle"`
	FooterTiktokUrl                  string     `gorm:"column:footerTiktokUrl"`
	JadwalHeaderSubtitle             string     `gorm:"column:jadwalHeaderSubtitle"`
	TentangTaglineQuote              string     `gorm:"column:tentangTaglineQuote"`
	TentangDescParagraph1            string     `gorm:"column:tentangDescParagraph1"`
	TentangDescParagraph2            string     `gorm:"column:tentangDescParagraph2"`
	TentangVisiText                  string     `gorm:"column:tentangVisiText"`
	TentangMisiJSON                  *string    `gorm:"column:tentangMisiJSON"`
	TentangTujuanJSON                *string    `gorm:"column:tentangTujuanJSON"`
	TentangKegiatanJSON              *string    `gorm:"column:tentangKegiatanJSON"`
	PesertaEarlyBirdDateRange        string     `gorm:"column:pesertaEarlyBirdDateRange"`
	PesertaRegulerDateRange          string     `gorm:"column:pesertaRegulerDateRange"`
	PesertaHargaNonSemarangEarlyBird int        `gorm:"column:pesertaHargaNonSemarangEarlyBird"`
	PesertaHargaNonSemarangReguler   int        `gorm:"column:pesertaHargaNonSemarangReguler"`
	PesertaHargaSemarangEarlyBird    int        `gorm:"column:pesertaHargaSemarangEarlyBird"`
	PesertaHargaSemarangReguler      int        `gorm:"column:pesertaHargaSemarangReguler"`
	PesertaBankName                  string     `gorm:"column:pesertaBankName"`
	PesertaBankAccountNumber         string     `gorm:"column:pesertaBankAccountNumber"`
	PesertaBankAccountHolder         string     `gorm:"column:pesertaBankAccountHolder"`
	PesertaGuidebookUrl              *string    `gorm:"column:pesertaGuidebookUrl"`
	PesertaGoogleFormUrl             string     `gorm:"column:pesertaGoogleFormUrl"`
	PesertaMapEmbedUrl               *string    `gorm:"column:pesertaMapEmbedUrl"`
	PanitiaIsOpen                    bool       `gorm:"column:panitiaIsOpen"`
	PanitiaClosedMessage             string     `gorm:"column:panitiaClosedMessage"`
	UpdatedDate                      *time.Time `gorm:"column:updatedDate"`
	UpdatedBy                        *int64     `gorm:"column:updatedBy"`
}

// GalleryImage merepresentasikan satu baris ms_rapimnas_gallery_image
// (galeri foto Beranda).
type GalleryImage struct {
	ID        int64  `gorm:"column:id;primaryKey;autoIncrement"`
	ImageUrl  string `gorm:"column:imageUrl"`
	SortOrder int    `gorm:"column:sortOrder"`
}

// HomeCard merepresentasikan satu baris ms_rapimnas_home_card (4 teaser
// "Rangkaian Kegiatan" di Beranda).
type HomeCard struct {
	ID          int64  `gorm:"column:id;primaryKey;autoIncrement"`
	IconKey     string `gorm:"column:iconKey"`
	Title       string `gorm:"column:title"`
	Description string `gorm:"column:description"`
	SortOrder   int    `gorm:"column:sortOrder"`
}

// RundownDay merepresentasikan satu baris ms_rapimnas_rundown_day (satu hari
// dalam jadwal 4-hari). ID kadang dipakai rapimnas_service/rapimnas_repository
// sebagai correlationID sementara saat membangun ulang koleksi dari
// UpdateRequest — lihat rapimnas_repository_impl.go.
type RundownDay struct {
	ID        int64  `gorm:"column:id;primaryKey;autoIncrement"`
	DayLabel  string `gorm:"column:dayLabel"`
	DateText  string `gorm:"column:dateText"`
	SortOrder int    `gorm:"column:sortOrder"`
}

// RundownEvent merepresentasikan satu baris ms_rapimnas_rundown_event (satu
// acara dalam satu hari, FK ke RundownDay lewat DayID).
type RundownEvent struct {
	ID          int64  `gorm:"column:id;primaryKey;autoIncrement"`
	DayID       int64  `gorm:"column:dayID"`
	Time        string `gorm:"column:time"`
	Title       string `gorm:"column:title"`
	Description string `gorm:"column:description"`
	Venue       string `gorm:"column:venue"`
	SortOrder   int    `gorm:"column:sortOrder"`
}

// Resource merepresentasikan satu baris ms_rapimnas_resource (daftar unduhan
// Arsip & Dokumen). IsVisible=false dipakai menyembunyikan entri dari
// endpoint publik tanpa menghapus baris (lihat rapimnas_service GetPublic).
type Resource struct {
	ID          int64  `gorm:"column:id;primaryKey;autoIncrement"`
	Title       string `gorm:"column:title"`
	Description string `gorm:"column:description"`
	IconKey     string `gorm:"column:iconKey"`
	Url         string `gorm:"column:url"`
	ButtonLabel string `gorm:"column:buttonLabel"`
	IsVisible   bool   `gorm:"column:isVisible"`
	SortOrder   int    `gorm:"column:sortOrder"`
}

// PickupLocation merepresentasikan satu baris ms_rapimnas_pickup_location
// (titik penjemputan di halaman Pendaftaran Peserta).
type PickupLocation struct {
	ID          int64  `gorm:"column:id;primaryKey;autoIncrement"`
	Name        string `gorm:"column:name"`
	Type        string `gorm:"column:type"`
	Description string `gorm:"column:description"`
	MapLink     string `gorm:"column:mapLink"`
	SortOrder   int    `gorm:"column:sortOrder"`
}

// Contact merepresentasikan satu baris ms_rapimnas_contact — menampung baik
// CP Pendaftaran Peserta (ContactTypePesertaCP) maupun kontak WA Footer
// (ContactTypeFooterWA) dalam satu tabel, dibedakan lewat ContactType.
type Contact struct {
	ID          int64  `gorm:"column:id;primaryKey;autoIncrement"`
	ContactType string `gorm:"column:contactType"`
	Name        string `gorm:"column:name"`
	PhoneNumber string `gorm:"column:phoneNumber"`
	SortOrder   int    `gorm:"column:sortOrder"`
}

// Aggregate menampung baris singleton beserta seluruh child collection —
// bentuk tukar-menukar data Repository.Get/Save. RundownDays/RundownEvents
// disimpan sebagai dua slice flat (bukan nested) karena model wajib murni
// data tanpa GORM association tag; penyusunan nested (event di bawah
// harinya) terjadi di rapimnas_service saat membangun DTO respons — lihat
// catatan "Design decisions" di kepala plan ini.
type Aggregate struct {
	Setting         Setting
	GalleryImages   []GalleryImage
	HomeCards       []HomeCard
	RundownDays     []RundownDay
	RundownEvents   []RundownEvent
	Resources       []Resource
	PickupLocations []PickupLocation
	Contacts        []Contact
}
