// Package upload menyimpan berkas gambar yang diunggah pengguna CMS ke disk
// lokal dan membentuk URL publiknya. Dipakai oleh modul article & news agar
// "Gambar Utama" bisa diisi lewat unggah berkas langsung, bukan hanya tempel URL.
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

// MaxFileSize adalah batas ukuran berkas gambar yang diterima (5 MB).
const MaxFileSize = 5 << 20

var allowedExt = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true}

// Uploader menyimpan berkas gambar ke direktori lokal dan mengembalikan URL publiknya.
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

// SaveImage memvalidasi ekstensi & ukuran berkas, menyimpannya dengan nama
// acak (menghindari tabrakan nama & path traversal dari nama asli), lalu
// mengembalikan URL publik yang bisa langsung dipakai sebagai articleImage/newsImage.
func (u *Uploader) SaveImage(fh *multipart.FileHeader) (string, error) {
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if !allowedExt[ext] {
		return "", fmt.Errorf("format berkas tidak didukung (hanya jpg, jpeg, png, webp, gif)")
	}
	if fh.Size > MaxFileSize {
		return "", fmt.Errorf("ukuran berkas melebihi 5MB")
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
