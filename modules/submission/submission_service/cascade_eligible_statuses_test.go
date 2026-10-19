package submission_service

import (
	"testing"

	"fsldk-api/constants"
)

// cascadeEligibleStatuses must include every status except the three
// dead-ends that publish-cascade deliberately skips (TechSpec DL-2): DRAFT
// (nothing meaningful to reset yet) and CANCELLED/REJECTED (already final).
func TestCascadeEligibleStatuses_ExcludesDraftCancelledRejected(t *testing.T) {
	excluded := map[string]bool{
		constants.SubmissionStatusDraft:     true,
		constants.SubmissionStatusCancelled: true,
		constants.SubmissionStatusRejected:  true,
	}
	for _, s := range cascadeEligibleStatuses {
		if excluded[s] {
			t.Errorf("cascadeEligibleStatuses contains %q, which must be excluded", s)
		}
	}
}

func TestCascadeEligibleStatuses_IncludesPublishedAndActive(t *testing.T) {
	want := []string{constants.SubmissionStatusPublished, constants.SubmissionStatusActive}
	for _, w := range want {
		found := false
		for _, s := range cascadeEligibleStatuses {
			if s == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("cascadeEligibleStatuses missing %q", w)
		}
	}
}
