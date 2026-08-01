// Package apperror menyediakan tipe error terstruktur untuk seluruh aplikasi.
// Setiap error membawa kode response internal, kode status HTTP, pesan yang
// aman ditampilkan ke klien, serta daftar detail validasi (opsional).
package apperror

import (
	"net/http"

	"fsldk-api/constants"
)

// FieldError merepresentasikan satu kesalahan validasi pada sebuah field.
type FieldError struct {
	Attribute string `json:"attribute"`
	Field     string `json:"field,omitempty"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Row       int    `json:"row,omitempty"`
}

// AppError adalah tipe error utama aplikasi.
type AppError struct {
	HTTPStatus int
	Code       string
	Message    string
	Fields     []FieldError
}

// Error memenuhi interface error.
func (e *AppError) Error() string { return e.Message }

// New membuat AppError baru.
func New(httpStatus int, code, message string) *AppError {
	return &AppError{HTTPStatus: httpStatus, Code: code, Message: message}
}

// Validation membuat error validasi (400) beserta detail field.
func Validation(message string, fields []FieldError) *AppError {
	if message == "" {
		message = "Validation Error"
	}
	return &AppError{HTTPStatus: http.StatusBadRequest, Code: constants.CodeValidationError, Message: message, Fields: fields}
}

// BadRequest membuat error 400 sederhana.
func BadRequest(message string) *AppError {
	return New(http.StatusBadRequest, constants.CodeValidationError, message)
}

// Unauthorized membuat error 401.
func Unauthorized(message string) *AppError {
	if message == "" {
		message = "Unauthorized"
	}
	return New(http.StatusUnauthorized, constants.CodeUnauthorized, message)
}

// Forbidden membuat error 403.
func Forbidden(message string) *AppError {
	if message == "" {
		message = "Forbidden"
	}
	return New(http.StatusForbidden, constants.CodeForbidden, message)
}

// EmailNotVerified membuat error 403 dengan kode khusus EMAIL_NOT_VERIFIED.
func EmailNotVerified() *AppError {
	return New(http.StatusForbidden, constants.CodeEmailNotVerified, "Email belum terverifikasi. Silakan verifikasi email Anda terlebih dahulu.")
}

// NotFound membuat error 404.
func NotFound(message string) *AppError {
	if message == "" {
		message = "Data tidak ditemukan"
	}
	return New(http.StatusNotFound, constants.CodeNotFound, message)
}

// Conflict membuat error 409.
func Conflict(message string) *AppError {
	return New(http.StatusConflict, constants.CodeConflict, message)
}

// Unprocessable membuat error 422 (pelanggaran aturan bisnis).
func Unprocessable(message string) *AppError {
	return New(http.StatusUnprocessableEntity, constants.CodeUnprocessable, message)
}

// TooManyRequests membuat error 429.
func TooManyRequests(message string) *AppError {
	if message == "" {
		message = "Terlalu banyak permintaan. Silakan coba beberapa saat lagi."
	}
	return New(http.StatusTooManyRequests, constants.CodeTooManyRequest, message)
}

// Internal membuat error 500. Pesan detail teknis tidak dibocorkan ke klien.
func Internal(message string) *AppError {
	if message == "" {
		message = "Terjadi kesalahan pada server"
	}
	return New(http.StatusInternalServerError, constants.CodeUnknownError, message)
}
