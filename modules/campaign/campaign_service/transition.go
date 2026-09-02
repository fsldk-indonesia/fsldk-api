package campaign_service

import "fsldk-api/constants"

// validTransitions memetakan status asal ke himpunan status tujuan yang sah.
// Revisi (2026-08-30): alur submission/review dihapus sesuai permintaan
// product owner — campaign kini murni CRUD + permission gate, tanpa langkah
// "diajukan lalu ditinjau orang lain". DRAFT langsung bisa dipublish oleh
// siapapun pemegang izin kantong_amal.campaign.publish. PUBLISHED/PAUSED
// juga langsung bisa diarsipkan (sebelumnya hanya lewat COMPLETED, yang
// tidak pernah benar-benar dipicu proses manapun — Archive() jadi tidak
// pernah bisa dipakai; bug lama ditutup sekalian di sini). COMPLETED dan
// EXPIRED tetap didaftarkan untuk proses sistem (deadline lewat, target
// tercapai) meski belum dibangun di fase ini.
var validTransitions = map[string]map[string]bool{
	constants.CampaignStatusDraft: {
		constants.CampaignStatusPublished: true,
	},
	constants.CampaignStatusPublished: {
		constants.CampaignStatusPaused:    true,
		constants.CampaignStatusCompleted: true,
		constants.CampaignStatusExpired:   true,
		constants.CampaignStatusArchived:  true,
	},
	constants.CampaignStatusPaused: {
		constants.CampaignStatusPublished: true,
		constants.CampaignStatusArchived:  true,
	},
	constants.CampaignStatusCompleted: {
		constants.CampaignStatusArchived: true,
	},
}

// isValidTransition menentukan apakah campaign boleh berpindah dari status
// from ke status to.
func isValidTransition(from, to string) bool {
	next, ok := validTransitions[from]
	if !ok {
		return false
	}
	return next[to]
}
