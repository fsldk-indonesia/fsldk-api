// Package upload menyimpan berkas (gambar/dokumen) yang diunggah pengguna
// CMS ke disk lokal dan membentuk URL publiknya. Dipakai oleh modul article
// & news agar "Gambar Utama" bisa diisi lewat unggah berkas langsung (bukan
// hanya tempel URL), oleh modul article untuk lampiran PDF artikel, dan oleh
// modul comment untuk lampiran gambar pada komentar.
package upload

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"fsldk-api/base/security"
)

// MaxImageSize adalah batas ukuran berkas gambar yang diterima (5 MB).
const MaxImageSize = 5 << 20

// MaxDocumentSize adalah batas ukuran berkas dokumen (PDF) yang diterima (20 MB).
const MaxDocumentSize = 20 << 20

var allowedImageExt = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true}
var allowedDocumentExt = map[string]bool{".pdf": true, ".docx": true, ".xlsx": true}

// Uploader menyimpan berkas ke direktori lokal dan mengembalikan URL publiknya.
type Uploader struct {
	dir     string
	baseURL string
}

// NewUploader membuat Uploader. dir adalah folder tujuan penyimpanan berkas
// (relatif terhadap direktori kerja proses), baseURL adalah URL publik backend
// (mis. cfg.AppURL) yang di depannya dipasang path statis "/uploads".
func NewUploader(dir, baseURL string) *Uploader {
	return &Uploader{dir: dir, baseURL: strings.TrimRight(baseURL, "/")}
}

// SaveImage memvalidasi ekstensi & ukuran berkas gambar lalu menyimpannya,
// mengembalikan URL publik yang bisa langsung dipakai sebagai articleImage/newsImage.
func (u *Uploader) SaveImage(fh *multipart.FileHeader) (string, error) {
	return u.save(fh, allowedImageExt, MaxImageSize, "format berkas tidak didukung (hanya jpg, jpeg, png, webp, gif)", "ukuran berkas melebihi 5MB")
}

// SaveDocument memvalidasi ekstensi & ukuran berkas dokumen (PDF/DOCX/XLSX)
// lalu menyimpannya, mengembalikan URL publik yang bisa dipakai sebagai
// articlePdf maupun dokumen pendukung submission.
func (u *Uploader) SaveDocument(fh *multipart.FileHeader) (string, error) {
	return u.save(fh, allowedDocumentExt, MaxDocumentSize, "format berkas tidak didukung (hanya pdf, docx, xlsx)", "ukuran berkas melebihi 20MB")
}

// save memvalidasi ekstensi & ukuran berkas, menyimpannya dengan nama acak
// (menghindari tabrakan nama & path traversal dari nama asli), lalu
// mengembalikan URL publiknya.
func (u *Uploader) save(fh *multipart.FileHeader, allowedExt map[string]bool, maxSize int64, extErrMsg, sizeErrMsg string) (string, error) {
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if !allowedExt[ext] {
		return "", fmt.Errorf("%s", extErrMsg)
	}
	if fh.Size > maxSize {
		return "", fmt.Errorf("%s", sizeErrMsg)
	}
	if err := os.MkdirAll(u.dir, 0755); err != nil {
		return "", err
	}

	token, err := security.RandomToken(16)
	if err != nil {
		return "", err
	}
	name := token + ext
	dst := filepath.Join(u.dir, name)

	src, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, src); err != nil {
		return "", err
	}

	return u.baseURL + "/uploads/" + name, nil
}

// DeleteFile removes a previously uploaded file given its public URL (the
// value SaveImage/SaveDocument returned). Idempotent: a missing file is not
// treated as an error, so callers can use it as best-effort cleanup.
func (u *Uploader) DeleteFile(publicURL string) error {
	name := path.Base(publicURL)
	if name == "" || name == "." || name == "/" {
		return nil
	}
	// publicURL segments are plain hex tokens (see save()), so no
	// percent-decoding is needed — this just guards against a stray query
	// string in case a caller passes a full request URL instead.
	if parsed, err := url.Parse(publicURL); err == nil {
		name = path.Base(parsed.Path)
	}

	err := os.Remove(filepath.Join(u.dir, name))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// FileSize returns the size in bytes of a previously uploaded file given its
// public URL. Read-only helper for module-specific size rules that need to be
// enforced after the shared upload endpoint has already accepted the file
// (which caps at MaxDocumentSize / MaxImageSize).
func (u *Uploader) FileSize(publicURL string) (int64, error) {
	info, err := os.Stat(u.LocalPath(publicURL))
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// LocalPath resolves a public URL (the value SaveImage/SaveDocument returned)
// to the file's path on disk. Read-only helper for callers that need to serve
// the file themselves (e.g. with a custom Content-Disposition filename).
func (u *Uploader) LocalPath(publicURL string) string {
	name := path.Base(publicURL)
	if parsed, err := url.Parse(publicURL); err == nil {
		name = path.Base(parsed.Path)
	}
	return filepath.Join(u.dir, name)
}
