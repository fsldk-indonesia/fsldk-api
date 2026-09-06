package dynamicform_service

import (
	"context"
	"encoding/json"
	"log"
	"time"
)

// Draft retention windows, measured from the draft's last update:
//   - staged files (uploaded images/attachments) are purged after 3 days
//   - the whole draft, text answers included, is purged after 7 days
const (
	draftFileTTL = 3 * 24 * time.Hour
	draftTTL     = 7 * 24 * time.Hour
)

// SweepStaleDrafts enforces the two retention windows above. Idempotent and safe
// to run row by row: stripping an already-stripped draft is a no-op.
func (s *ServiceImpl) SweepStaleDrafts(ctx context.Context) error {
	now := time.Now()
	fileCutoff := now.Add(-draftFileTTL)
	draftCutoff := now.Add(-draftTTL)

	// One query for everything older than the shorter (file) window; the longer
	// (whole-draft) window is a subset filtered in memory.
	drafts, err := s.repo.StaleDrafts(ctx, fileCutoff)
	if err != nil {
		return err
	}

	removedDrafts, strippedFiles := 0, 0
	for _, d := range drafts {
		m := map[string]json.RawMessage{}
		if json.Unmarshal([]byte(d.AnswersJSON), &m) != nil {
			m = nil
		}

		// Older than 7 days: drop the draft and every staged file it still holds.
		if d.UpdatedDate.Before(draftCutoff) {
			for _, v := range m {
				if url, _, _, _, ok := stagedFileEntry(v); ok && url != "" {
					_ = s.uploader.DeleteFile(url)
				}
			}
			if err := s.repo.DeleteDraftByID(ctx, d.DraftID); err == nil {
				removedDrafts++
			}
			continue
		}

		// 3–7 days old: delete staged files but keep the text answers.
		changed := false
		for key, v := range m {
			url, _, _, _, ok := stagedFileEntry(v)
			if !ok {
				continue
			}
			if url != "" {
				_ = s.uploader.DeleteFile(url)
			}
			delete(m, key)
			changed = true
		}
		if changed {
			if b, mErr := json.Marshal(m); mErr == nil {
				if err := s.repo.SetDraftAnswers(ctx, d.DraftID, string(b)); err == nil {
					strippedFiles++
				}
			}
		}
	}
	if removedDrafts > 0 || strippedFiles > 0 {
		log.Printf("[DYNAMICFORM] sweep: removed %d stale draft(s), stripped files from %d", removedDrafts, strippedFiles)
	}
	return nil
}

// SweepOrphanUploads is intentionally conservative. pkg/upload writes every
// module's files into the same assets/uploads/ directory under random tokens
// with no per-module prefix, so a directory scan cannot tell a dynamicform
// upload from an article image. Rather than risk deleting another module's
// files, orphan cleanup is delegated to SweepStaleDrafts (which knows the exact
// URLs it staged) and to the per-response file cleanup on delete/edit. This
// hook stays as a no-op unless a dedicated upload sub-directory is introduced.
func (s *ServiceImpl) SweepOrphanUploads(ctx context.Context) error {
	_ = ctx
	return nil
}
