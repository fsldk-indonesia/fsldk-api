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

// CityCount is the active-LDK count for one city/kabupaten — mirrors
// ProvinceCount, used to populate the "Kota/Kabupaten" directory filter
// data-driven (distinct values that actually exist), same rule as province.
type CityCount struct {
	CityName string `json:"cityName"`
	Count    int    `json:"count"`
}

// NetworkStatsResponse is the aggregate public "Statistik Jaringan" payload.
type NetworkStatsResponse struct {
	TotalPuskomnas   int             `json:"totalPuskomnas"`
	TotalPuskomda    int             `json:"totalPuskomda"`
	TotalLDK         int             `json:"totalLDK"`
	TotalActiveKader int             `json:"totalActiveKader"`
	ByProvince       []ProvinceCount `json:"byProvince"`
	ByLevel          []LevelCount    `json:"byLevel"`
	ByCity           []CityCount     `json:"byCity"`
}

// DirectoryEntry is one organization row in the public network directory —
// PhotoURL is the organization's logo, shown when set (fallback to a generic
// icon is a frontend concern, not this DTO's). ParentOrganizationID/Name are
// nil for the root Puskomnas (no parent); the frontend uses them to render
// the directory as a Puskomnas -> Puskomda -> LDK tree instead of a flat
// list, mirroring ms_organization.parentOrganizationID. ContactEmail/
// ContactPhone/WebsiteURL are shown on the public LDK card too — this was a
// deliberate product decision (contact details are meant to be discoverable
// for the public directory), not an oversight. LevelCode/LevelLabel are only
// populated for the organization's CURRENT, PUBLISHED Levelisasi result
// (tr_levelisasi_result.isCurrent=1 AND isPublished=1) — unpublished or
// non-LDK organizations simply get empty strings here, same rule as
// statistic_repository.LevelDistribution().
type DirectoryEntry struct {
	OrganizationID         int64   `json:"organizationID"`
	OrganizationTypeCode   string  `json:"organizationTypeCode"`
	OrganizationName       string  `json:"organizationName"`
	OrganizationCode       string  `json:"organizationCode"`
	ProvinceName           string  `json:"provinceName,omitempty"`
	CityName               string  `json:"cityName,omitempty"`
	ContactEmail           string  `json:"contactEmail,omitempty"`
	ContactPhone           string  `json:"contactPhone,omitempty"`
	WebsiteURL             string  `json:"websiteURL,omitempty"`
	PhotoURL               string  `json:"photoURL,omitempty"`
	ParentOrganizationID   *int64  `json:"parentOrganizationID,omitempty"`
	ParentOrganizationName *string `json:"parentOrganizationName,omitempty"`
	LevelCode              string  `json:"levelCode,omitempty"`
	LevelLabel             string  `json:"levelLabel,omitempty"`
}

// DirectoryFilter groups Directory()'s optional multi-select filters into one
// struct instead of positional params — avoids a positional-argument mix-up
// each time another filter column is added, same convention as
// article_dto.PublicFilter. Every field is matched via SQL IN (empty/nil
// slice means "no filter on this field").
type DirectoryFilter struct {
	OrganizationTypeCodes []string // filter "Tipe Unit" (LDK/PUSKOMDA/PUSKOMNAS)
	ProvinceNames         []string // filter "Provinsi"
	CityNames             []string // filter "Kota/Kabupaten"
	// ParentOrganizationIDs matches o.organizationID OR o.parentOrganizationID
	// — filter "Puskomda Induk" selecting a Puskomda returns the Puskomda row
	// itself PLUS its LDKs, not just the LDKs.
	ParentOrganizationIDs []int64
	// LevelCodes matches tr_levelisasi_result.levelCode (current+published,
	// already scoped by the JOIN condition in Directory()) — filter "Level
	// Levelisasi". Puskomda/Puskomnas never have a level, so they're
	// naturally excluded whenever this filter is active.
	LevelCodes []string
}
