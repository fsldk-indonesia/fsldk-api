// Package rapimnas_dto memuat DTO request/response modul rapimnas. Seluruhnya
// murni struct data (tanpa function/method).
package rapimnas_dto

// GalleryImageDTO adalah satu entri galeri foto Beranda.
type GalleryImageDTO struct {
	ImageUrl  string `json:"imageUrl" validate:"required,max=500"`
	SortOrder int    `json:"sortOrder"`
}

// HomeCardDTO adalah satu teaser "Rangkaian Kegiatan" di Beranda.
type HomeCardDTO struct {
	IconKey     string `json:"iconKey" validate:"required,max=50"`
	Title       string `json:"title" validate:"required,max=255"`
	Description string `json:"description" validate:"required"`
	SortOrder   int    `json:"sortOrder"`
}

// RundownEventDTO adalah satu acara dalam satu hari jadwal.
type RundownEventDTO struct {
	Time        string `json:"time" validate:"max=100"`
	Title       string `json:"title" validate:"required,max=255"`
	Description string `json:"description" validate:"required"`
	Venue       string `json:"venue" validate:"max=255"`
	SortOrder   int    `json:"sortOrder"`
}

// RundownDayDTO adalah satu hari jadwal, membawa seluruh acaranya (Events)
// ter-nested — bentuk ini yang dipertukarkan lewat API; penyimpanannya di DB
// tetap dua tabel terpisah (lihat rapimnas_model.Aggregate).
type RundownDayDTO struct {
	DayLabel  string             `json:"dayLabel" validate:"required,max=100"`
	DateText  string             `json:"dateText" validate:"required,max=100"`
	SortOrder int                `json:"sortOrder"`
	Events    []RundownEventDTO  `json:"events" validate:"omitempty,dive"`
}

// ResourceDTO adalah satu entri unduhan Arsip & Dokumen.
type ResourceDTO struct {
	Title       string `json:"title" validate:"required,max=255"`
	Description string `json:"description" validate:"required"`
	IconKey     string `json:"iconKey" validate:"required,max=50"`
	Url         string `json:"url" validate:"required,max=500"`
	ButtonLabel string `json:"buttonLabel" validate:"required,max=100"`
	IsVisible   bool   `json:"isVisible"`
	SortOrder   int    `json:"sortOrder"`
}

// PickupLocationDTO adalah satu titik penjemputan Pendaftaran Peserta.
type PickupLocationDTO struct {
	Name        string `json:"name" validate:"required,max=255"`
	Type        string `json:"type" validate:"required,max=50"`
	Description string `json:"description"`
	MapLink     string `json:"mapLink" validate:"required,max=500"`
	SortOrder   int    `json:"sortOrder"`
}

// ContactDTO adalah satu kontak (CP Peserta ATAU WA Footer, dibedakan
// ContactType).
type ContactDTO struct {
	ContactType string `json:"contactType" validate:"required,oneof=peserta_cp footer_wa"`
	Name        string `json:"name" validate:"required,max=100"`
	PhoneNumber string `json:"phoneNumber" validate:"required,max=30"`
	SortOrder   int    `json:"sortOrder"`
}

// PublicResponse adalah payload lengkap untuk 6 halaman publik /rapimnas —
// satu panggilan API mengambil semuanya (hero, countdown, feature cards, cta,
// footer, tentang, peserta, panitia) plus 6 child collection, rundown
// bernested events-nya. SENGAJA tidak membawa updatedDate/updatedBy (lihat
// CMSResponse) — mengikuti pola welcomepopup_dto.PublicResponse. Resources
// pada respons ini HANYA yang IsVisible=true (lihat rapimnas_service).
type PublicResponse struct {
	HeroBadgeText       string `json:"heroBadgeText"`
	HeroTitle           string `json:"heroTitle"`
	HeroDateRangeText   string `json:"heroDateRangeText"`
	HeroTaglineQuote    string `json:"heroTaglineQuote"`
	HeroImageUrl        string `json:"heroImageUrl"`
	CountdownTargetDate string `json:"countdownTargetDate"`

	Feature1IconKey string `json:"feature1IconKey"`
	Feature1Title   string `json:"feature1Title"`
	Feature1Desc    string `json:"feature1Desc"`
	Feature2IconKey string `json:"feature2IconKey"`
	Feature2Title   string `json:"feature2Title"`
	Feature2Desc    string `json:"feature2Desc"`

	CtaTitle          string `json:"ctaTitle"`
	CtaDescription    string `json:"ctaDescription"`
	CtaButtonLabel    string `json:"ctaButtonLabel"`
	CtaMascotImageUrl string `json:"ctaMascotImageUrl"`

	FooterContactEmail  string `json:"footerContactEmail"`
	FooterCopyrightText string `json:"footerCopyrightText"`
	FooterIgHandle      string `json:"footerIgHandle"`
	FooterIgUrl         string `json:"footerIgUrl"`
	FooterTiktokHandle  string `json:"footerTiktokHandle"`
	FooterTiktokUrl     string `json:"footerTiktokUrl"`

	JadwalHeaderSubtitle string `json:"jadwalHeaderSubtitle"`

	TentangTaglineQuote   string   `json:"tentangTaglineQuote"`
	TentangDescParagraph1 string   `json:"tentangDescParagraph1"`
	TentangDescParagraph2 string   `json:"tentangDescParagraph2"`
	TentangVisiText       string   `json:"tentangVisiText"`
	TentangMisi           []string `json:"tentangMisi"`
	TentangTujuan         []string `json:"tentangTujuan"`
	TentangKegiatan       []string `json:"tentangKegiatan"`

	PesertaEarlyBirdDateRange        string `json:"pesertaEarlyBirdDateRange"`
	PesertaRegulerDateRange          string `json:"pesertaRegulerDateRange"`
	PesertaHargaNonSemarangEarlyBird int    `json:"pesertaHargaNonSemarangEarlyBird"`
	PesertaHargaNonSemarangReguler   int    `json:"pesertaHargaNonSemarangReguler"`
	PesertaHargaSemarangEarlyBird    int    `json:"pesertaHargaSemarangEarlyBird"`
	PesertaHargaSemarangReguler      int    `json:"pesertaHargaSemarangReguler"`
	PesertaBankName                  string `json:"pesertaBankName"`
	PesertaBankAccountNumber         string `json:"pesertaBankAccountNumber"`
	PesertaBankAccountHolder         string `json:"pesertaBankAccountHolder"`
	PesertaGuidebookUrl              string `json:"pesertaGuidebookUrl"`
	PesertaGoogleFormUrl             string `json:"pesertaGoogleFormUrl"`
	PesertaMapEmbedUrl               string `json:"pesertaMapEmbedUrl"`

	PanitiaIsOpen        bool   `json:"panitiaIsOpen"`
	PanitiaClosedMessage string `json:"panitiaClosedMessage"`

	GalleryImages   []GalleryImageDTO   `json:"galleryImages"`
	HomeCards       []HomeCardDTO       `json:"homeCards"`
	Rundown         []RundownDayDTO     `json:"rundownDays"`
	Resources       []ResourceDTO       `json:"resources"`
	PickupLocations []PickupLocationDTO `json:"pickupLocations"`
	Contacts        []ContactDTO        `json:"contacts"`
}

// CMSResponse sama seperti PublicResponse ditambah metadata audit. Berbeda
// dari PublicResponse, Resources di sini membawa SELURUH baris termasuk yang
// IsVisible=false — supaya admin bisa mengedit/mengaktifkan kembali arsip
// yang disembunyikan (lihat rapimnas_service.toCMSResponse).
type CMSResponse struct {
	PublicResponse
	UpdatedDate string `json:"updatedDate"`
	UpdatedBy   *int64 `json:"updatedBy"`
}

// UpdateRequest adalah body PUT /rapimnas-setup — payload nested utuh,
// disubmit sekali lewat satu tombol Simpan (lihat design spec, "one Simpan
// button"). SortOrder pada tiap item child SENGAJA diabaikan oleh
// rapimnas_service — urutan final ditentukan oleh urutan array yang
// dikirim (lihat rapimnas_service.toAggregate), bukan nilai integer ini;
// field ini tetap ada untuk simetri dengan respons.
type UpdateRequest struct {
	HeroBadgeText       string `json:"heroBadgeText" validate:"required,max=100"`
	HeroTitle           string `json:"heroTitle" validate:"required,max=255"`
	HeroDateRangeText   string `json:"heroDateRangeText" validate:"required,max=100"`
	HeroTaglineQuote    string `json:"heroTaglineQuote" validate:"required,max=500"`
	HeroImageUrl        string `json:"heroImageUrl" validate:"omitempty,max=500"`
	CountdownTargetDate string `json:"countdownTargetDate" validate:"required"`

	Feature1IconKey string `json:"feature1IconKey" validate:"required,max=50"`
	Feature1Title   string `json:"feature1Title" validate:"required,max=255"`
	Feature1Desc    string `json:"feature1Desc" validate:"required"`
	Feature2IconKey string `json:"feature2IconKey" validate:"required,max=50"`
	Feature2Title   string `json:"feature2Title" validate:"required,max=255"`
	Feature2Desc    string `json:"feature2Desc" validate:"required"`

	CtaTitle          string `json:"ctaTitle" validate:"required,max=255"`
	CtaDescription    string `json:"ctaDescription" validate:"required"`
	CtaButtonLabel    string `json:"ctaButtonLabel" validate:"required,max=100"`
	CtaMascotImageUrl string `json:"ctaMascotImageUrl" validate:"omitempty,max=500"`

	FooterContactEmail  string `json:"footerContactEmail" validate:"required,email"`
	FooterCopyrightText string `json:"footerCopyrightText" validate:"required,max=255"`
	FooterIgHandle      string `json:"footerIgHandle" validate:"required,max=100"`
	FooterIgUrl         string `json:"footerIgUrl" validate:"required,url"`
	FooterTiktokHandle  string `json:"footerTiktokHandle" validate:"required,max=100"`
	FooterTiktokUrl     string `json:"footerTiktokUrl" validate:"required,url"`

	JadwalHeaderSubtitle string `json:"jadwalHeaderSubtitle" validate:"required,max=500"`

	TentangTaglineQuote   string   `json:"tentangTaglineQuote" validate:"required,max=500"`
	TentangDescParagraph1 string   `json:"tentangDescParagraph1" validate:"required"`
	TentangDescParagraph2 string   `json:"tentangDescParagraph2" validate:"required"`
	TentangVisiText       string   `json:"tentangVisiText" validate:"required"`
	TentangMisi           []string `json:"tentangMisi" validate:"omitempty,dive,required"`
	TentangTujuan         []string `json:"tentangTujuan" validate:"omitempty,dive,required"`
	TentangKegiatan       []string `json:"tentangKegiatan" validate:"omitempty,dive,required"`

	PesertaEarlyBirdDateRange        string `json:"pesertaEarlyBirdDateRange" validate:"required,max=100"`
	PesertaRegulerDateRange          string `json:"pesertaRegulerDateRange" validate:"required,max=100"`
	PesertaHargaNonSemarangEarlyBird int    `json:"pesertaHargaNonSemarangEarlyBird" validate:"min=0"`
	PesertaHargaNonSemarangReguler   int    `json:"pesertaHargaNonSemarangReguler" validate:"min=0"`
	PesertaHargaSemarangEarlyBird    int    `json:"pesertaHargaSemarangEarlyBird" validate:"min=0"`
	PesertaHargaSemarangReguler      int    `json:"pesertaHargaSemarangReguler" validate:"min=0"`
	PesertaBankName                  string `json:"pesertaBankName" validate:"required,max=100"`
	PesertaBankAccountNumber         string `json:"pesertaBankAccountNumber" validate:"required,max=50"`
	PesertaBankAccountHolder         string `json:"pesertaBankAccountHolder" validate:"required,max=100"`
	PesertaGuidebookUrl              string `json:"pesertaGuidebookUrl" validate:"omitempty,url"`
	PesertaGoogleFormUrl             string `json:"pesertaGoogleFormUrl" validate:"required,url"`
	PesertaMapEmbedUrl               string `json:"pesertaMapEmbedUrl" validate:"omitempty,url"`

	PanitiaIsOpen        bool   `json:"panitiaIsOpen"`
	PanitiaClosedMessage string `json:"panitiaClosedMessage" validate:"required"`

	GalleryImages   []GalleryImageDTO   `json:"galleryImages" validate:"omitempty,dive"`
	HomeCards       []HomeCardDTO       `json:"homeCards" validate:"omitempty,dive"`
	Rundown         []RundownDayDTO     `json:"rundownDays" validate:"omitempty,dive"`
	Resources       []ResourceDTO       `json:"resources" validate:"omitempty,dive"`
	PickupLocations []PickupLocationDTO `json:"pickupLocations" validate:"omitempty,dive"`
	Contacts        []ContactDTO        `json:"contacts" validate:"omitempty,dive"`
}
