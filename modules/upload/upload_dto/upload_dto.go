// Package upload_dto memuat DTO response modul upload. Murni struct data.
package upload_dto

// Response adalah representasi hasil unggah berkas gambar.
type Response struct {
	URL string `json:"url"`
}
