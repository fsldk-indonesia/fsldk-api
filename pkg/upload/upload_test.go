package upload

import (
	"bytes"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"
)

func TestThumbFileName(t *testing.T) {
	cases := map[string]string{
		"abc123.jpg":  "abc123_thumb.jpg",
		"abc123.jpeg": "abc123_thumb.jpeg",
		"abc123.png":  "abc123_thumb.png",
		"abc123.webp": "abc123_thumb.webp",
	}
	for input, want := range cases {
		if got := thumbFileName(input); got != want {
			t.Errorf("thumbFileName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestDeleteFile_RemovesMainAndThumb(t *testing.T) {
	dir := t.TempDir()
	u := NewUploader(dir, "http://localhost:8080")

	mainPath := filepath.Join(dir, "sometoken.jpg")
	thumbPath := filepath.Join(dir, "sometoken_thumb.jpg")
	if err := os.WriteFile(mainPath, []byte("main"), 0644); err != nil {
		t.Fatalf("failed to seed main file: %v", err)
	}
	if err := os.WriteFile(thumbPath, []byte("thumb"), 0644); err != nil {
		t.Fatalf("failed to seed thumb file: %v", err)
	}

	if err := u.DeleteFile("http://localhost:8080/uploads/sometoken.jpg"); err != nil {
		t.Fatalf("DeleteFile returned error: %v", err)
	}

	if _, err := os.Stat(mainPath); !os.IsNotExist(err) {
		t.Errorf("expected main file to be removed, stat err = %v", err)
	}
	if _, err := os.Stat(thumbPath); !os.IsNotExist(err) {
		t.Errorf("expected thumb file to be removed, stat err = %v", err)
	}
}

func TestDeleteFile_MissingThumbIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	u := NewUploader(dir, "http://localhost:8080")

	mainPath := filepath.Join(dir, "sometoken.jpg")
	if err := os.WriteFile(mainPath, []byte("main"), 0644); err != nil {
		t.Fatalf("failed to seed main file: %v", err)
	}

	// No thumb file seeded (e.g. pre-existing upload from before this
	// feature) — deleting must still succeed.
	if err := u.DeleteFile("http://localhost:8080/uploads/sometoken.jpg"); err != nil {
		t.Fatalf("DeleteFile returned error when thumb is missing: %v", err)
	}
}

func newImageFileHeader(t *testing.T, filename string, data []byte) *multipart.FileHeader {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	h.Set("Content-Type", "image/jpeg")
	part, err := w.CreatePart(h)
	if err != nil {
		t.Fatalf("failed to create form part: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("failed to write form part: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, "/", &buf)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	if err := req.ParseMultipartForm(32 << 20); err != nil {
		t.Fatalf("failed to parse multipart form: %v", err)
	}
	_, fh, err := req.FormFile("file")
	if err != nil {
		t.Fatalf("failed to read form file: %v", err)
	}
	return fh
}

func TestSaveImage_WritesResizedMainAndThumbFiles(t *testing.T) {
	dir := t.TempDir()
	u := NewUploader(dir, "http://localhost:8080")

	original := encodeTestJPEG(t, 3000, 2000)
	fh := newImageFileHeader(t, "photo.jpg", original)

	url, err := u.SaveImage(fh)
	if err != nil {
		t.Fatalf("SaveImage returned error: %v", err)
	}

	mainLocal := u.LocalPath(url)
	mainBytes, err := os.ReadFile(mainLocal)
	if err != nil {
		t.Fatalf("main file not found on disk at %s: %v", mainLocal, err)
	}
	mainCfg, err := jpeg.DecodeConfig(bytes.NewReader(mainBytes))
	if err != nil {
		t.Fatalf("main file is not a valid jpeg: %v", err)
	}
	if mainCfg.Width != maxMainWidth {
		t.Errorf("expected main width %d, got %d", maxMainWidth, mainCfg.Width)
	}

	thumbLocal := filepath.Join(filepath.Dir(mainLocal), thumbFileName(filepath.Base(mainLocal)))
	thumbBytes, err := os.ReadFile(thumbLocal)
	if err != nil {
		t.Fatalf("thumb file not found on disk at %s: %v", thumbLocal, err)
	}
	thumbCfg, err := jpeg.DecodeConfig(bytes.NewReader(thumbBytes))
	if err != nil {
		t.Fatalf("thumb file is not a valid jpeg: %v", err)
	}
	if thumbCfg.Width != maxThumbWidth {
		t.Errorf("expected thumb width %d, got %d", maxThumbWidth, thumbCfg.Width)
	}
}
