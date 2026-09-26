// Package dashboard_dto memuat DTO response modul dashboard.
package dashboard_dto

import "time"

// NoteEntry adalah satu catatan terbaru pada riwayat status submission.
type NoteEntry struct {
	Note        string    `json:"note"`
	CreatedDate time.Time `json:"createdDate"`
}

// LDKSummary adalah ringkasan dashboard untuk caller bertipe LDK.
type LDKSummary struct {
	SubmissionStatus string      `json:"submissionStatus"`
	LastUpdatedDate  *time.Time  `json:"lastUpdatedDate,omitempty"`
	LevelCode        string      `json:"levelCode,omitempty"`
	LevelLabel       string      `json:"levelLabel,omitempty"`
	KaderPending     int         `json:"kaderPending"`
	KaderActive      int         `json:"kaderActive"`
	RecentNotes      []NoteEntry `json:"recentNotes"`
}

// StatusCounts adalah rekap jumlah LDK per bucket status Levelisasi.
type StatusCounts struct {
	BelumMengisi       int `json:"belumMengisi"`
	MenungguVerifikasi int `json:"menungguVerifikasi"`
	PerluRevisi        int `json:"perluRevisi"`
	Terverifikasi      int `json:"terverifikasi"`
}

// PuskomdaSummary adalah ringkasan dashboard untuk caller bertipe Puskomda.
type PuskomdaSummary struct {
	TotalLDK int `json:"totalLDK"`
	StatusCounts
	TotalKaderAktif int `json:"totalKaderAktif"`
}

// LevelCount adalah jumlah LDK per level hasil levelisasi saat ini.
type LevelCount struct {
	LevelCode  string `json:"levelCode"`
	LevelLabel string `json:"levelLabel"`
	Count      int    `json:"count"`
}

// PuskomdaBreakdown adalah rekap per Puskomda untuk widget sebaran nasional.
type PuskomdaBreakdown struct {
	OrganizationID   int64  `json:"organizationID"`
	OrganizationName string `json:"organizationName"`
	TotalLDK         int    `json:"totalLDK"`
	KaderAktif       int    `json:"kaderAktif"`
}

// PuskomnasSummary adalah ringkasan dashboard untuk caller bertipe Puskomnas.
type PuskomnasSummary struct {
	TotalLDKNasional int `json:"totalLDKNasional"`
	StatusCounts
	LevelEstablishedCount   int                 `json:"levelEstablishedCount"`
	TotalPuskomda           int                 `json:"totalPuskomda"`
	TotalKaderAktifNasional int                 `json:"totalKaderAktifNasional"`
	LevelDistribution       []LevelCount        `json:"levelDistribution"`
	PerPuskomda             []PuskomdaBreakdown `json:"perPuskomda"`
}

// UtamaSummary adalah ringkasan dashboard khusus CMS Utama (FSLDK) — metrik
// administrasi sistem (pengguna, konten) SATU set, PLUS ringkasan jaringan
// Levelisasi nasional (StatusCounts + Network*) SET LAIN. Sebelumnya bagian
// kedua ini sengaja tidak dimuat di sini (lihat miss-development-prompt-3.md
// poin 5) supaya CMS Utama tidak menduplikasi dashboard Puskomnas — namun
// atas permintaan eksplisit, Super Admin butuh melihat kondisi jaringan
// nasional ini juga langsung dari CMS Utama tanpa pindah shell ke
// cms-puskomnas. Nilainya dihitung persis sama seperti cabang Puskomnas di
// Summary() (StatusBuckets/CountLDK/dst. dengan parentOrganizationID nil —
// scope nasional), BUKAN duplikasi query terpisah.
//
// Setiap field administrasi-sistem di sini berpasangan dengan satu modul
// yang tampil di sidebar shell CMS Utama (lihat app.routes.ts fsldk-web,
// children dari path 'cms') — field baru ditambah di sini SEKALIGUS
// repo/service count-nya supaya modul baru tidak diam-diam hilang dari
// widget statistik dashboard.
type UtamaSummary struct {
	TotalUsers            int     `json:"totalUsers"`
	TotalRoles            int     `json:"totalRoles"`
	TotalNews             int     `json:"totalNews"`
	TotalArticles         int     `json:"totalArticles"`
	TotalEvents           int     `json:"totalEvents"`
	TotalSchedules        int     `json:"totalSchedules"`
	TotalGalleries        int     `json:"totalGalleries"`
	TotalStructures       int     `json:"totalStructures"`
	TotalCatalogBooks     int     `json:"totalCatalogBooks"`
	TotalDynamicForms     int     `json:"totalDynamicForms"`
	TotalGoodsProducts    int     `json:"totalGoodsProducts"`
	TotalFinanceFormats   int     `json:"totalFinanceFormats"`
	TotalCampaigns        int     `json:"totalCampaigns"`
	TotalDonationCollected float64 `json:"totalDonationCollected"`
	TotalComments         int     `json:"totalComments"`
	TotalShortlinks       int     `json:"totalShortlinks"`
	TotalQrcodes          int     `json:"totalQrcodes"`
	TotalSubscribers      int     `json:"totalSubscribers"`
	UnreadContactMessages int     `json:"unreadContactMessages"`
	PendingJobs           int     `json:"pendingJobs"`

	// Ringkasan jaringan Levelisasi nasional — sama persis dengan yang
	// dipakai PuskomnasSummary (lihat komentar di atas).
	StatusCounts
	NetworkTotalLDK          int                 `json:"networkTotalLDK"`
	NetworkTotalPuskomda     int                 `json:"networkTotalPuskomda"`
	NetworkKaderAktif        int                 `json:"networkKaderAktif"`
	NetworkLevelDistribution []LevelCount        `json:"networkLevelDistribution"`
	NetworkPerPuskomda       []PuskomdaBreakdown `json:"networkPerPuskomda"`
}

// Summary adalah response GET /dashboard/summary — hanya satu dari Utama/LDK/
// Puskomda/Puskomnas yang terisi, sesuai konteks shell yang meminta (tier).
type Summary struct {
	OrganizationTypeCode string            `json:"organizationTypeCode"`
	Utama                *UtamaSummary     `json:"utama,omitempty"`
	LDK                  *LDKSummary       `json:"ldk,omitempty"`
	Puskomda             *PuskomdaSummary  `json:"puskomda,omitempty"`
	Puskomnas            *PuskomnasSummary `json:"puskomnas,omitempty"`
}
