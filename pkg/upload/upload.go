// Package upload menyimpan berkas (gambar/dokumen) yang diunggah pengguna
// CMS ke disk lokal dan membentuk URL publiknya. Dipakai oleh modul article
// & news agar "Gambar Utama" bisa diisi lewat unggah berkas langsung (bukan
// hanya tempel URL), dan oleh modul article untuk lampiran PDF artikel.
package upload

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
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
