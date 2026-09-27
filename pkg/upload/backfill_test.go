package upload

import (
	"bytes"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

func TestBackfillThumbnails_SkipsFilesThatAlreadyHaveThumb(t *testing.T) {
	dir := t.TempDir()
	u := NewUploader(dir, "http://localhost:8080")

	mainPath := filepath.Join(dir, "main.jpg")
	thumbPath := filepath.Join(dir, "main_thumb.jpg")
	writeFile(t, mainPath, []byte("original-main-bytes"))
	writeFile(t, thumbPath, []byte("existing-thumb-bytes"))

	result, err := u.BackfillThumbnails()
	if err != nil {
		t.Fatalf("BackfillThumbnails returned error: %v", err)
	}

	if len(result.Processed) != 0 {
		t.Errorf("expected nothing processed, got %v", result.Processed)
	}
	if len(result.Skipped) != 1 || result.Skipped[0] != "main.jpg" {
		t.Errorf("expected main.jpg to be skipped, got %v", result.Skipped)
	}

	thumbBytes, _ := os.ReadFile(thumbPath)
	if string(thumbBytes) != "existing-thumb-bytes" {
		t.Errorf("expected existing thumb to be left untouched, got %q", thumbBytes)
	}
}

func TestBackfillThumbnails_GeneratesThumbAndResizesOversizedMain(t *testing.T) {
	dir := t.TempDir()
	u := NewUploader(dir, "http://localhost:8080")

	mainPath := filepath.Join(dir, "big.jpg")
	writeFile(t, mainPath, encodeTestJPEG(t, 3000, 2000))

	result, err := u.BackfillThumbnails()
	if err != nil {
		t.Fatalf("BackfillThumbnails returned error: %v", err)
	}
	if len(result.Processed) != 1 || result.Processed[0] != "big.jpg" {
		t.Errorf("expected big.jpg to be processed, got %v", result.Processed)
	}

	mainBytes, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("failed to read main file: %v", err)
	}
	mainCfg, err := jpeg.DecodeConfig(bytes.NewReader(mainBytes))
	if err != nil {
		t.Fatalf("main file is not a valid jpeg: %v", err)
	}
	if mainCfg.Width != maxMainWidth {
		t.Errorf("expected main to be resized to width %d, got %d", maxMainWidth, mainCfg.Width)
	}

	thumbBytes, err := os.ReadFile(filepath.Join(dir, "big_thumb.jpg"))
	if err != nil {
		t.Fatalf("thumb file was not created: %v", err)
	}
	thumbCfg, err := jpeg.DecodeConfig(bytes.NewReader(thumbBytes))
	if err != nil {
		t.Fatalf("thumb file is not a valid jpeg: %v", err)
	}
	if thumbCfg.Width != maxThumbWidth {
		t.Errorf("expected thumb width %d, got %d", maxThumbWidth, thumbCfg.Width)
	}
}

func TestBackfillThumbnails_CreatesThumbCopyForWebpWithoutResizing(t *testing.T) {
	dir := t.TempDir()
	u := NewUploader(dir, "http://localhost:8080")

	mainPath := filepath.Join(dir, "photo.webp")
	original := []byte("not-a-real-webp-but-thats-fine-for-this-test")
	writeFile(t, mainPath, original)

	result, err := u.BackfillThumbnails()
	if err != nil {
		t.Fatalf("BackfillThumbnails returned error: %v", err)
	}
	if len(result.Processed) != 1 || result.Processed[0] != "photo.webp" {
		t.Errorf("expected photo.webp to be processed, got %v", result.Processed)
	}

	mainBytes, _ := os.ReadFile(mainPath)
	if !bytes.Equal(mainBytes, original) {
		t.Errorf("expected webp main file to stay untouched")
	}

	thumbBytes, err := os.ReadFile(filepath.Join(dir, "photo_thumb.webp"))
	if err != nil {
		t.Fatalf("thumb file was not created: %v", err)
	}
	if !bytes.Equal(thumbBytes, original) {
		t.Errorf("expected webp thumb to be a plain copy of the original")
	}
}

func TestBackfillThumbnails_IgnoresOrphanThumbAndNonImageFiles(t *testing.T) {
	dir := t.TempDir()
	u := NewUploader(dir, "http://localhost:8080")

	writeFile(t, filepath.Join(dir, "notes.txt"), []byte("not an image"))
	writeFile(t, filepath.Join(dir, "orphan_thumb.jpg"), []byte("orphan thumb, no matching main"))

	result, err := u.BackfillThumbnails()
	if err != nil {
		t.Fatalf("BackfillThumbnails returned error: %v", err)
	}
	if len(result.Processed) != 0 || len(result.Skipped) != 0 {
		t.Errorf("expected nothing processed or skipped, got processed=%v skipped=%v", result.Processed, result.Skipped)
	}
	if _, err := os.Stat(filepath.Join(dir, "orphan_thumb_thumb.jpg")); !os.IsNotExist(err) {
		t.Errorf("orphan thumb file must not be treated as a main image")
	}
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}
