// Package statistic_dto contains data transfer objects for the public
// "Statistik Jaringan" feature — aggregate, anonymized counts about the
// LDK/Puskomda/Puskomnas network, safe to expose without authentication.
package statistic_dto

// TypeCount is the active-organization count for one organization type.
type TypeCount struct {
	OrganizationTypeCode string `json:"organizationTypeCode"`
	Count                int    `json:"count"`
}

// ProvinceCount is the active-LDK count for one province.
type ProvinceCount struct {
	ProvinceName string `json:"provinceName"`
	Count        int    `json:"count"`
}

// LevelCount is the current, published Levelisasi LDK count for one level.
type LevelCount struct {
	LevelCode  string `json:"levelCode"`
	LevelLabel string `json:"levelLabel"`
	Count      int    `json:"count"`
}

// NetworkStatsResponse is the aggregate public "Statistik Jaringan" payload.
type NetworkStatsResponse struct {
	TotalPuskomnas   int             `json:"totalPuskomnas"`
	TotalPuskomda    int             `json:"totalPuskomda"`
	TotalLDK         int             `json:"totalLDK"`
	TotalActiveKader int             `json:"totalActiveKader"`
	ByProvince       []ProvinceCount `json:"byProvince"`
	ByLevel          []LevelCount    `json:"byLevel"`
}

// DirectoryEntry is one organization row in the public network directory —
// PhotoURL is the organization's logo, shown when set (fallback to a generic
// icon is a frontend concern, not this DTO's).
type DirectoryEntry struct {
	OrganizationID       int64  `json:"organizationID"`
	OrganizationTypeCode string `json:"organizationTypeCode"`
	OrganizationName     string `json:"organizationName"`
	ProvinceName         string `json:"provinceName,omitempty"`
	CityName             string `json:"cityName,omitempty"`
	PhotoURL             string `json:"photoURL,omitempty"`
}
