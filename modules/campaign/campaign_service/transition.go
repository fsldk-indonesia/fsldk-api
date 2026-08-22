package campaign_service

import "fsldk-api/constants"

// validTransitions memetakan status asal ke himpunan status tujuan yang sah.
// COMPLETED dan EXPIRED hanya dicapai lewat proses sistem (deadline lewat,
// target tercapai — belum dibangun di fase ini) sehingga belum ada handler
// yang memicunya, tapi tetap didaftarkan di sini supaya validasi transisi
// sudah konsisten begitu proses tersebut dibangun tanpa perlu mengubah tabel
// ini lagi nanti.
var validTransitions = map[string]map[string]bool{
	constants.CampaignStatusDraft: {
		constants.CampaignStatusSubmitted: true,
	},
	constants.CampaignStatusRevisionRequested: {
		constants.CampaignStatusSubmitted: true,
	},
	constants.CampaignStatusSubmitted: {
		constants.CampaignStatusApproved:          true,
		constants.CampaignStatusRevisionRequested: true,
		constants.CampaignStatusRejected:          true,
	},
	constants.CampaignStatusApproved: {
		constants.CampaignStatusPublished: true,
	},
	constants.CampaignStatusPublished: {
		constants.CampaignStatusPaused:    true,
		constants.CampaignStatusCompleted: true,
		constants.CampaignStatusExpired:   true,
	},
	constants.CampaignStatusPaused: {
		constants.CampaignStatusPublished: true,
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
