// Package setting_service memuat logika bisnis modul setting.
package setting_service

import (
	"context"

	"fsldk-api/modules/setting/setting_dto"
)

// Service adalah kontrak logika bisnis App Settings.
type Service interface {
	List(ctx context.Context) ([]setting_dto.Response, error)
	Update(ctx context.Context, id int64, req setting_dto.UpdateRequest, actorID int64) (setting_dto.Response, error)
	// GetValue mengembalikan nilai satu setting berdasarkan group+key. "" bila
	// tidak ditemukan/kosong — BUKAN error, supaya pemanggil (mis.
	// shortlinkrequest_service) tetap bisa jalan meski belum dikonfigurasi.
	GetValue(ctx context.Context, group, key string) (string, error)
}
