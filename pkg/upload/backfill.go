package upload

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

// BackfillResult summarizes a BackfillThumbnails run.
type BackfillResult struct {
	// Processed lists main image filenames that got a new thumb generated
	// (and, for jpg/png, had their main file capped in place if it was
	// larger than maxMainWidth).
	Processed []string
	// Skipped lists main image filenames that already had a thumb and were
	// left untouched.
	Skipped []string
	// Failed maps a main image filename to the error that stopped it from
	// being processed (e.g. unreadable file). A failure here does not stop
	// the rest of the scan.
	Failed map[string]error
}

// BackfillThumbnails scans the upload directory for images saved before the
// thumbnail feature existed and generates the missing "_thumb" variant for
// each. Safe to run repeatedly — any file that already has a "_thumb"
// sibling, or is itself a "_thumb" file, is left alone.
func (u *Uploader) BackfillThumbnails() (BackfillResult, error) {
	result := BackfillResult{Failed: map[string]error{}}

	entries, err := os.ReadDir(u.dir)
	if err != nil {
		return result, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if !allowedImageExt[ext] {
			continue
		}
		if strings.HasSuffix(strings.TrimSuffix(name, ext), "_thumb") {
			continue // this IS a thumb file, not a main image to process
		}

		mainPath := filepath.Join(u.dir, name)
		thumbPath := filepath.Join(u.dir, thumbFileName(name))

		if _, statErr := os.Stat(thumbPath); statErr == nil {
			result.Skipped = append(result.Skipped, name)
			continue
		}

		original, readErr := os.ReadFile(mainPath)
		if readErr != nil {
			result.Failed[name] = readErr
			continue
		}

		main, thumb := generateVariants(original, ext)

		if !bytes.Equal(main, original) {
			if writeErr := writeAtomic(mainPath, main); writeErr != nil {
				result.Failed[name] = writeErr
				continue
			}
		}
		if writeErr := writeAtomic(thumbPath, thumb); writeErr != nil {
			result.Failed[name] = writeErr
			continue
		}

		result.Processed = append(result.Processed, name)
	}

	return result, nil
}

// writeAtomic writes data to a temp file in the same directory then renames
// it into place, so an interrupted write never leaves a half-written file at
// the real path — important here since main files can be overwritten
// in-place with their resized version.
func writeAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
