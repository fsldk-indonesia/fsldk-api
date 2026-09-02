package campaign_service

import (
	"testing"

	"fsldk-api/constants"
)

var campaignAllStatuses = []string{
	constants.CampaignStatusDraft,
	constants.CampaignStatusSubmitted,
	constants.CampaignStatusRevisionRequested,
	constants.CampaignStatusApproved,
	constants.CampaignStatusPublished,
	constants.CampaignStatusPaused,
	constants.CampaignStatusCompleted,
	constants.CampaignStatusRejected,
	constants.CampaignStatusArchived,
	constants.CampaignStatusExpired,
}

// TestIsValidTransition_ExhaustiveMatrix menguji seluruh 10x10 kombinasi
// status asal/tujuan (100 pasangan) terhadap daftar transisi sah yang
// didefinisikan independen dari validTransitions — memastikan tidak ada
// transisi tak sengaja terbuka/tertutup akibat salah ketik di map asli.
// Revisi (2026-08-30): alur submission/review dihapus — DRAFT langsung ke
// PUBLISHED, dan PUBLISHED/PAUSED bisa langsung diarsipkan (bug lama
// tertutup sekalian: Archive() sebelumnya tidak pernah bisa dipakai karena
// hanya valid dari COMPLETED, yang tidak pernah dipicu proses manapun).
func TestIsValidTransition_ExhaustiveMatrix(t *testing.T) {
	valid := map[[2]string]bool{
		{constants.CampaignStatusDraft, constants.CampaignStatusPublished}:     true,
		{constants.CampaignStatusPublished, constants.CampaignStatusPaused}:    true,
		{constants.CampaignStatusPublished, constants.CampaignStatusCompleted}: true,
		{constants.CampaignStatusPublished, constants.CampaignStatusExpired}:   true,
		{constants.CampaignStatusPublished, constants.CampaignStatusArchived}:  true,
		{constants.CampaignStatusPaused, constants.CampaignStatusPublished}:    true,
		{constants.CampaignStatusPaused, constants.CampaignStatusArchived}:     true,
		{constants.CampaignStatusCompleted, constants.CampaignStatusArchived}:  true,
	}

	for _, from := range campaignAllStatuses {
		for _, to := range campaignAllStatuses {
			want := valid[[2]string{from, to}]
			if got := isValidTransition(from, to); got != want {
				t.Errorf("isValidTransition(%s, %s) = %v, want %v", from, to, got, want)
			}
		}
	}
}

func TestIsValidTransition_UnknownStatusRejected(t *testing.T) {
	if isValidTransition("UNKNOWN", constants.CampaignStatusPublished) {
		t.Fatal("expected transition from unknown status to be rejected")
	}
	if isValidTransition(constants.CampaignStatusDraft, "UNKNOWN") {
		t.Fatal("expected transition to unknown status to be rejected")
	}
}
