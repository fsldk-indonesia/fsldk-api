package statistic_service_test

import (
	"context"
	"testing"

	"fsldk-api/base/dto"
	"fsldk-api/modules/statistic/statistic_dto"
	"fsldk-api/modules/statistic/statistic_service"
)

type mockRepository struct {
	byType      []statistic_dto.TypeCount
	byProvince  []statistic_dto.ProvinceCount
	byLevel     []statistic_dto.LevelCount
	activeKader int
	directory   []statistic_dto.DirectoryEntry
}

func (m *mockRepository) CountByType(ctx context.Context) ([]statistic_dto.TypeCount, error) {
	return m.byType, nil
}
func (m *mockRepository) CountByProvince(ctx context.Context) ([]statistic_dto.ProvinceCount, error) {
	return m.byProvince, nil
}
func (m *mockRepository) LevelDistribution(ctx context.Context) ([]statistic_dto.LevelCount, error) {
	return m.byLevel, nil
}
func (m *mockRepository) CountActiveKader(ctx context.Context) (int, error) {
	return m.activeKader, nil
}
func (m *mockRepository) Directory(ctx context.Context, q dto.ListQuery, organizationTypeCode, provinceName string) ([]statistic_dto.DirectoryEntry, int, error) {
	return m.directory, len(m.directory), nil
}

func TestStatisticService_NetworkStats_MapsCountsByType(t *testing.T) {
	repo := &mockRepository{
		byType: []statistic_dto.TypeCount{
			{OrganizationTypeCode: "PUSKOMNAS", Count: 1},
			{OrganizationTypeCode: "PUSKOMDA", Count: 12},
			{OrganizationTypeCode: "LDK", Count: 340},
		},
		byProvince:  []statistic_dto.ProvinceCount{{ProvinceName: "DKI Jakarta", Count: 20}},
		byLevel:     []statistic_dto.LevelCount{{LevelCode: "MADYA", LevelLabel: "Madya", Count: 50}},
		activeKader: 1200,
	}
	svc := statistic_service.NewService(repo)

	resp, err := svc.NetworkStats(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if resp.TotalPuskomnas != 1 {
		t.Errorf("expected TotalPuskomnas=1, got %d", resp.TotalPuskomnas)
	}
	if resp.TotalPuskomda != 12 {
		t.Errorf("expected TotalPuskomda=12, got %d", resp.TotalPuskomda)
	}
	if resp.TotalLDK != 340 {
		t.Errorf("expected TotalLDK=340, got %d", resp.TotalLDK)
	}
	if resp.TotalActiveKader != 1200 {
		t.Errorf("expected TotalActiveKader=1200, got %d", resp.TotalActiveKader)
	}
	if len(resp.ByProvince) != 1 || len(resp.ByLevel) != 1 {
		t.Errorf("expected byProvince/byLevel to pass through unchanged, got %+v / %+v", resp.ByProvince, resp.ByLevel)
	}
}

func TestStatisticService_NetworkStats_IgnoresUnknownType(t *testing.T) {
	repo := &mockRepository{
		byType: []statistic_dto.TypeCount{{OrganizationTypeCode: "SOMETHING_ELSE", Count: 99}},
	}
	svc := statistic_service.NewService(repo)

	resp, err := svc.NetworkStats(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if resp.TotalPuskomnas != 0 || resp.TotalPuskomda != 0 || resp.TotalLDK != 0 {
		t.Errorf("expected all totals to remain 0 for an unrecognized type, got %+v", resp)
	}
}

func TestStatisticService_Directory_PassesThrough(t *testing.T) {
	repo := &mockRepository{
		directory: []statistic_dto.DirectoryEntry{
			{OrganizationID: 1, OrganizationName: "LDK Contoh", OrganizationTypeCode: "LDK"},
		},
	}
	svc := statistic_service.NewService(repo)

	items, total, err := svc.Directory(context.Background(), dto.ListQuery{Page: 1, Limit: 20}, "LDK", "")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("expected 1 directory entry, got total=%d len=%d", total, len(items))
	}
}
