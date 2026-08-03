// Package upload_service memuat logika bisnis modul upload.
package upload_service

import "mime/multipart"

// Service adalah kontrak logika bisnis unggah berkas gambar.
type Service interface {
	SaveImage(fh *multipart.FileHeader) (string, error)
}
