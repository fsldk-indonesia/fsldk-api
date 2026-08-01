// Package httphelper menyediakan format response HTTP yang seragam untuk
// seluruh endpoint, mengikuti standar response pada TechSpec (bagian 20):
// { path, timestamp, status, code, message, result, errors }.
package httphelper

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"fsldk-api/base/apperror"
	"fsldk-api/constants"

	"github.com/gin-gonic/gin"
)

// Response adalah amplop (envelope) standar untuk seluruh response.
type Response struct {
	Path      string                 `json:"path"`
	Timestamp string                 `json:"timestamp"`
	Status    string                 `json:"status"`
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Result    interface{}            `json:"result"`
	Errors    []apperror.FieldError  `json:"errors"`
}

// Pagination adalah struktur hasil untuk endpoint list dengan paginasi.
type Pagination[T any] struct {
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
	Count int    `json:"count"`
	Data  []T    `json:"data"`
	Prev  string `json:"prev"`
	Next  string `json:"next"`
}

func fullPath(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s%s", scheme, c.Request.Host, c.Request.RequestURI)
}

// Success menuliskan response sukses (200) dengan pesan & data.
func Success(c *gin.Context, message string, result interface{}) {
	write(c, http.StatusOK, constants.CodeSuccess, msgOr(message, "Ok"), result, nil)
}

// Created menuliskan response 201.
func Created(c *gin.Context, message string, result interface{}) {
	write(c, http.StatusCreated, constants.CodeSuccess, msgOr(message, "Created"), result, nil)
}

// Error menuliskan response error berdasarkan tipe error yang diterima.
// AppError dipetakan ke status & kode-nya; error lain menjadi 500.
func Error(c *gin.Context, err error) {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		write(c, appErr.HTTPStatus, appErr.Code, appErr.Message, nil, appErr.Fields)
		return
	}
	write(c, http.StatusInternalServerError, constants.CodeUnknownError, "Terjadi kesalahan pada server", nil, nil)
}

func write(c *gin.Context, httpStatus int, code, message string, result interface{}, fields []apperror.FieldError) {
	status := "fail"
	if httpStatus >= 200 && httpStatus < 300 {
		status = "ok"
	}
	c.JSON(httpStatus, Response{
		Path:      fullPath(c),
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Status:    status,
		Code:      code,
		Message:   message,
		Result:    result,
		Errors:    fields,
	})
	c.Abort()
}

func msgOr(msg, fallback string) string {
	if msg == "" {
		return fallback
	}
	return msg
}

// BuildPagination menyusun struktur paginasi lengkap dengan URL prev/next.
func BuildPagination[T any](c *gin.Context, data []T, count, page, limit int) Pagination[T] {
	base := fullPathWithoutQuery(c)
	prev := ""
	next := ""
	if page > 1 {
		prev = fmt.Sprintf("%s?page=%d&limit=%d", base, page-1, limit)
	}
	if page*limit < count {
		next = fmt.Sprintf("%s?page=%d&limit=%d", base, page+1, limit)
	}
	if data == nil {
		data = []T{}
	}
	return Pagination[T]{Page: page, Limit: limit, Count: count, Data: data, Prev: prev, Next: next}
}

func fullPathWithoutQuery(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s%s", scheme, c.Request.Host, c.Request.URL.Path)
}
