package dashboard_service

import (
	"context"
	"testing"
	"time"

	"fsldk-api/modules/dashboard/dashboard_dto"
	"fsldk-api/modules/setting/setting_dto"
	"fsldk-api/modules/setting/setting_model"
)

// fakeDeadlineSettingService adalah implementasi setting_service.Service
// in-memory — hanya GetValue/GetValue2 yang berperilaku bermakna (dipakai
// deadlineAlerts), List/Update tidak pernah dipanggil di sini jadi no-op.
type fakeDeadlineSettingService struct {
	values  map[string]string // key -> tanggal "YYYY-MM-DD"
	values2 map[string]string // key -> estimasi biaya (digit polos)
}

func (f *fakeDeadlineSettingService) List(ctx context.Context) ([]setting_dto.Response, error) {
	return nil, nil
}
func (f *fakeDeadlineSettingService) Update(ctx context.Context, id int64, req setting_dto.UpdateRequest, actorID int64) (setting_dto.Response, error) {
	return setting_dto.Response{}, nil
}
func (f *fakeDeadlineSettingService) GetValue(ctx context.Context, group, key string) (string, error) {
	return f.values[key], nil
}
func (f *fakeDeadlineSettingService) GetValue2(ctx context.Context, group, key string) (string, error) {
	return f.values2[key], nil
}

// dateDaysFromNow menghitung tanggal persis N hari dari sekarang MEMAKAI
// basis truncation yang sama dengan deadlineAlerts() (today :=
// time.Now().Truncate(24*time.Hour)) — supaya daysRemaining hasil hitung
// fungsi production PERSIS sama dengan `days` di sini, apa pun timezone
// mesin yang menjalankan test (menghindari off-by-one akibat truncation).
func dateDaysFromNow(days int) string {
	today := time.Now().Truncate(24 * time.Hour)
	return today.AddDate(0, 0, days).Format("2006-01-02")
}

func hasAlertFor(alerts []dashboard_dto.DeadlineAlert, key string) bool {
	for _, a := range alerts {
		if a.SettingKey == key {
			return true
		}
	}
	return false
}

func TestDeadlineAlerts_HostingServer_IncludedWithin90Days(t *testing.T) {
	setting := &fakeDeadlineSettingService{values: map[string]string{
		setting_model.KeyDeadlineHostingServer: dateDaysFromNow(85),
	}}
	svc := &ServiceImpl{setting: setting}

	alerts := svc.deadlineAlerts(context.Background())

	if !hasAlertFor(alerts, setting_model.KeyDeadlineHostingServer) {
		t.Errorf("expected Hosting Server alert at 85 days remaining to be included, alerts = %+v", alerts)
	}
}

func TestDeadlineAlerts_HostingServer_ExcludedBeyond90Days(t *testing.T) {
	setting := &fakeDeadlineSettingService{values: map[string]string{
		setting_model.KeyDeadlineHostingServer: dateDaysFromNow(95),
	}}
	svc := &ServiceImpl{setting: setting}

	alerts := svc.deadlineAlerts(context.Background())

	if hasAlertFor(alerts, setting_model.KeyDeadlineHostingServer) {
		t.Errorf("expected Hosting Server alert at 95 days remaining to be EXCLUDED, alerts = %+v", alerts)
	}
}

func TestDeadlineAlerts_RentDomain_IncludedWithin90Days(t *testing.T) {
	setting := &fakeDeadlineSettingService{values: map[string]string{
		setting_model.KeyDeadlineRentDomain: dateDaysFromNow(90),
	}}
	svc := &ServiceImpl{setting: setting}

	alerts := svc.deadlineAlerts(context.Background())

	if !hasAlertFor(alerts, setting_model.KeyDeadlineRentDomain) {
		t.Errorf("expected Rent Domain alert at 90 days remaining to be included, alerts = %+v", alerts)
	}
}

func TestDeadlineAlerts_RentDomain_ExcludedBeyond90Days(t *testing.T) {
	setting := &fakeDeadlineSettingService{values: map[string]string{
		setting_model.KeyDeadlineRentDomain: dateDaysFromNow(91),
	}}
	svc := &ServiceImpl{setting: setting}

	alerts := svc.deadlineAlerts(context.Background())

	if hasAlertFor(alerts, setting_model.KeyDeadlineRentDomain) {
		t.Errorf("expected Rent Domain alert at 91 days remaining to be EXCLUDED, alerts = %+v", alerts)
	}
}

func TestDeadlineAlerts_WhatsAppIntegration_StillUses30DayThreshold(t *testing.T) {
	setting := &fakeDeadlineSettingService{values: map[string]string{
		setting_model.KeyDeadlineWhatsAppIntegration: dateDaysFromNow(60),
	}}
	svc := &ServiceImpl{setting: setting}

	alerts := svc.deadlineAlerts(context.Background())

	if hasAlertFor(alerts, setting_model.KeyDeadlineWhatsAppIntegration) {
		t.Errorf("WhatsApp Integration threshold must stay at 30 days (not extended to 90), alerts = %+v", alerts)
	}
}

func TestDeadlineAlerts_CertificateWebsite_StillUses30DayThreshold(t *testing.T) {
	setting := &fakeDeadlineSettingService{values: map[string]string{
		setting_model.KeyDeadlineCertificateWebsite: dateDaysFromNow(60),
	}}
	svc := &ServiceImpl{setting: setting}

	alerts := svc.deadlineAlerts(context.Background())

	if hasAlertFor(alerts, setting_model.KeyDeadlineCertificateWebsite) {
		t.Errorf("Certificate Website threshold must stay at 30 days (not extended to 90), alerts = %+v", alerts)
	}
}
