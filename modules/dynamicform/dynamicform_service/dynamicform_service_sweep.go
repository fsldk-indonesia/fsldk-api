package dynamicform_service

import (
	"context"
	"encoding/json"
	"log"
	"time"
)

// SweepStaleDrafts deletes drafts untouched for more than 7 days and best-effort
// removes their staged files from disk. Idempotent and safe to run concurrently
// (it operates row by row).
func (s *ServiceImpl) SweepStaleDrafts(ctx context.Context) error {
	cutoff := time.Now().AddDate(0, 0, -7)
	drafts, err := s.repo.StaleDrafts(ctx, cutoff)
	if err != nil {
		return err
	}
	removed := 0
	for _, d := range drafts {
		m := map[string]json.RawMessage{}
		if json.Unmarshal([]byte(d.AnswersJSON), &m) == nil {
			for _, v := range m {
				if url, _, _, _, ok := stagedFileEntry(v); ok && url != "" {
					_ = s.uploader.DeleteFile(url)
				}
			}
		}
		if err := s.repo.DeleteDraftByID(ctx, d.DraftID); err == nil {
			removed++
		}
	}
	if removed > 0 {
		log.Printf("[DYNAMICFORM] sweep: removed %d stale draft(s)", removed)
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
