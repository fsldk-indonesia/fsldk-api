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

// Summary adalah response GET /dashboard/summary — hanya satu dari LDK/Puskomda/
// Puskomnas yang terisi, sesuai organizationTypeCode caller.
type Summary struct {
	OrganizationTypeCode string            `json:"organizationTypeCode"`
	LDK                  *LDKSummary       `json:"ldk,omitempty"`
	Puskomda             *PuskomdaSummary  `json:"puskomda,omitempty"`
	Puskomnas            *PuskomnasSummary `json:"puskomnas,omitempty"`
}
